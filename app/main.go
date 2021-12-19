package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NamespaceFilePathEnv  string = "NAMESPACE_FILEPATH"
	PodNameFilePathEnv    string = "POD_NAME_FILEPATH"
	MessageAnnotationName string = "MESSAGE_ANNOTATION_NAME"
	BaseURLEnv            string = "BASE_URL"

	TypeLabel          string = "type"
	StatusLabel        string = "status"
	ReasonLabel        string = "reason"
	MessageAnnotation  string = "message"
	PingTimeAnnotation string = "ping-time"

	ServiceOffline string = "ServiceOffline"
	ServiceOnline  string = "ServiceOnline"

	PingSucceeded string = "PingSucceeded"
	PingFailed    string = "PingFailed"
)

type Result struct {
	Type     string
	Status   bool
	Reason   string
	Message  string
	PingTime string
}

func main() {
	logger := log.Default()

	logger.Println("Start WebPinger")
	pingURL, pingURLErr := getPingURL()
	if pingURLErr != nil {
		logger.Panic(pingURLErr)
	}
	logger.Println("Ping Url: ", pingURL)

	namespace, namespaceErr := getNamespace()
	if namespaceErr != nil {
		logger.Panic(namespaceErr)
	}
	logger.Println("Namespace:", namespace)

	podName, podNameErr := getPodName()
	if podNameErr != nil {
		logger.Panic(podNameErr)
	}
	logger.Println("Pod name:", podName)

	config, configErr := rest.InClusterConfig()
	if configErr != nil {
		logger.Panic(configErr)
	}

	clientset, clientsetErr := kubernetes.NewForConfig(config)
	if clientsetErr != nil {
		logger.Panic(clientsetErr)
	}

	pingResult, headers, body, pingErr := webPing(
		prepareHTTPClient(),
		pingURL,
		getTime,
	)
	if pingErr != nil {
		logger.Println("Error: ", pingErr)
	}
	logger.Println("Headers: ", headers)
	logger.Println("Response body: ", string(body))

	thisPod, getPodErr := clientset.
		CoreV1().
		Pods(namespace).
		Get(context.Background(), podName, metav1.GetOptions{})

	if getPodErr != nil {
		logger.Panic(getPodErr)
	}

	_, updatePodErr := updatePod(context.Background(), clientset, thisPod, pingResult)
	if updatePodErr != nil {
		logger.Panic(updatePodErr)
	}

	logger.Println("Completed WebPinger")
}

func getPingURL() (string, error) {
	baseURL := os.Getenv(BaseURLEnv)
	path := ""
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	pingURL, err := url.Parse(baseURL + path)
	return pingURL.String(), err
}

func getNamespace() (string, error) {
	path := os.Getenv(NamespaceFilePathEnv)
	if path == "" {
		path = "/etc/podinfo/namespace"
	}
	namespace, err := os.ReadFile(path)
	return string(namespace), err
}

func getPodName() (string, error) {
	path := os.Getenv(PodNameFilePathEnv)
	if path == "" {
		path = "/etc/podinfo/name"
	}
	name, err := os.ReadFile(path)
	return string(name), err
}

type timeGetter func() string

func getTime() string {
	now := time.Now().UTC()
	timeBody, err := metav1.NewTime(now).MarshalJSON()
	if err != nil {
		panic(err)
	}
	return string(timeBody)
}

func prepareHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

func webPing(
	client *http.Client,
	url string,
	getTime timeGetter,
) (result Result, headers http.Header, body []byte, err error) {
	response, err := client.Get(url)

	result = Result{
		Type:     ServiceOffline,
		Status:   false,
		Reason:   PingFailed,
		Message:  "",
		PingTime: getTime(),
	}

	if err != nil {
		return result, nil, nil, err
	}

	defer response.Body.Close()

	result.Type = ServiceOnline

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		result.Status = true
		result.Reason = PingSucceeded
	}

	headers = response.Header

	body, err = io.ReadAll(response.Body)
	if err != nil {
		result.Status = false
		result.Reason = PingFailed
	}
	result.Message = string(body)

	return result, headers, body, err
}

func updatePod(
	ctx context.Context,
	clientset *kubernetes.Clientset,
	pod *v1.Pod,
	updateData Result,
) (updatedPod *v1.Pod, updateErr error) {
	updatingPod := pod.DeepCopy()

	if updatingPod.Annotations == nil {
		updatingPod.Annotations = map[string]string{}
	}
	updatingPod.Annotations[MessageAnnotation] = updateData.Message
	updatingPod.Annotations[PingTimeAnnotation] = updateData.PingTime

	if updatingPod.Labels == nil {
		updatingPod.Labels = map[string]string{}
	}
	updatingPod.Labels[TypeLabel] = updateData.Type
	updatingPod.Labels[StatusLabel] = fmt.Sprintf("%v", updateData.Status)
	updatingPod.Labels[ReasonLabel] = updateData.Reason

	updatedPod, updateErr = clientset.
		CoreV1().
		Pods(updatingPod.GetNamespace()).
		Update(ctx, updatingPod, metav1.UpdateOptions{})

	return updatedPod, updateErr
}
