// Package load provides multithreading job and config creation that uses
// patchanalyzer to load vulnerabilities into the vulnerability database
package load

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"

	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	jobv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	cmv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/ichbinfrog/vulas-utils/pkg/connect"
	"github.com/ichbinfrog/vulas-utils/pkg/convert"
	"gopkg.in/yaml.v3"
)

// CVE represents a simplified vulnerability to be loaded
type CVE struct {
	Reference   string `yaml:"reference"`
	Repo        string `yaml:"repo"`
	Commit      string `yaml:"commit"`
	Description string `yaml:"description"`
	Links       string `yaml:"links"`
}

// Context represents the Context passed to execute the loading job
type Context struct {
	KubeConfig  string `yaml:"kubeconfig"`
	Concurrent  int    `yaml:"concurrent"`
	ReleaseName string `yaml:"release"`
	Namespace   string `yaml:"namespace"`
	Skip        bool   `yaml:"skip"`
	DryRun      bool   `yaml:"dryrun"`
	Source      string `yaml:"source"`
}

func getCVEList(source *string) ([]CVE, error) {
	if _, err := os.Stat(*source); err != nil {
		return nil, err
	}
	yamlFile, readErr := ioutil.ReadFile(*source)
	if readErr != nil {
		return nil, readErr
	}

	cveList := make(map[string][]CVE)
	if unmarshallErr := yaml.Unmarshal(yamlFile, cveList); unmarshallErr != nil {
		return nil, unmarshallErr
	}

	if cveList["bugs"] != nil {
		return cveList["bugs"], nil
	}

	return nil, fmt.Errorf("Malformed source file")
}

// SplitCVE splits loaded bugs into equal chunks to be distributed
func SplitCVE(context *Context) ([][]CVE, error) {
	cveList, err := getCVEList(&context.Source)
	if err != nil {
		return nil, err
	}

	var distributedCve [][]CVE
	chunkSize := (len(cveList) + context.Concurrent - 1) / context.Concurrent

	for i := 0; i < len(cveList); i += chunkSize {
		end := i + chunkSize
		if end > len(cveList) {
			end = len(cveList)
		}

		distributedCve = append(distributedCve, cveList[i:end])
	}

	return distributedCve, nil
}

// UploadBugs helps uploqd the bugs into the desired restbackend
func UploadBugs(context *Context, bugs [][]CVE) {
	var wg sync.WaitGroup
	clientset, connectErr := connect.GetClient(context.KubeConfig)
	if connectErr != nil {
		log.Fatal(connectErr)
	}
	jobClient := clientset.BatchV1().Jobs(context.Namespace)
	configClient := clientset.CoreV1().ConfigMaps(context.Namespace)

	for chunkID, cveList := range bugs {
		wg.Add(1)
		go func(context *Context, configClient cmv1.ConfigMapInterface, jobClient jobv1.JobInterface, bugs []CVE, chunkID int) {
			defer wg.Done()
			createConfigMap(context, configClient, bugs, chunkID)
			createJob(jobClient, chunkID)
		}(context, configClient, jobClient, cveList, chunkID)
		wg.Wait()
	}

	cleanUp(jobClient, configClient)
}

func getChunkName(chunkID int) string {
	return "bugs-loader-" + strconv.Itoa(chunkID)
}

func getBackendService(release *string) string {
	return *release + "restbackend-service:8091/backend"
}

func getLoaderCommand(bugs []CVE, context *Context) string {
	patcheval := `
#!/bin/sh
  `
	for _, bug := range bugs {
		patcheval = patcheval + fmt.Sprintf(`
java -Dvulas.shared.backend.serviceUrl=%s \
      -jar patch-analyzer-jar-with-dependencies.jar \
      -b %s \
      -r %s \
      -e %s \
      -desc %q \
      -links %q`, getBackendService(&context.ReleaseName), bug.Reference, bug.Repo, bug.Commit, bug.Description, bug.Links)

		if !context.DryRun {
			patcheval = patcheval + " -u"
		}

		if context.Skip {
			patcheval = patcheval + " -sie"
		}

		// Allows for continuous uninterrupted execution even if failure
		patcheval = patcheval + " || : "
	}

	return patcheval
}

func createConfigMap(context *Context, configClient cmv1.ConfigMapInterface, bugs []CVE, chunkID int) {
	configmap := &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: getChunkName(chunkID),
			Labels: map[string]string{
				"app.kubernetes.io/name":     "bugs-loader",
				"app.kubernetes.io/instance": strconv.Itoa(chunkID),
			},
		},
		Data: map[string]string{
			"patcheval.sh": getLoaderCommand(bugs, context),
		},
	}

	log.Println(configmap)
	_, err := configClient.Create(configmap)
	if err != nil {
		deleteErr := configClient.Delete(configmap.Name, &metav1.DeleteOptions{})
		if deleteErr != nil {
			log.Fatal(err, deleteErr)
		}
		log.Fatal(err)
	}
}

func cleanUp(jobClient jobv1.JobInterface, configClient cmv1.ConfigMapInterface) {
	jobList, listErr := jobClient.List(metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=bugs-loader",
	})

	if listErr != nil {
		log.Fatal(listErr)
	}

	for _, job := range jobList.Items {
		jobClient.Delete(job.Name, &metav1.DeleteOptions{})
	}

	configList, configListErr := configClient.List(metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=bugs-loader",
	})

	if configListErr != nil {
		log.Fatal(configListErr)
	}

	for _, cm := range configList.Items {
		configClient.Delete(cm.Name, &metav1.DeleteOptions{})
	}
}

func createJob(jobClient jobv1.JobInterface, chunkID int) {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: getChunkName(chunkID),
		},
		Spec: batchv1.JobSpec{
			Parallelism:           convert.Int32Ptr(1),
			ActiveDeadlineSeconds: convert.Int64Ptr(100),
			BackoffLimit:          convert.Int32Ptr(0),
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name":     "bugs-loader",
						"app.kubernetes.io/instance": strconv.Itoa(chunkID),
					},
				},
				Spec: apiv1.PodSpec{
					RestartPolicy: "Never",
					Volumes: []apiv1.Volume{
						{
							Name: getChunkName(chunkID),
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: getChunkName(chunkID),
									},
								},
							},
						},
					},
					Containers: []apiv1.Container{
						{
							Name:            "bugs-loader",
							Image:           "ichbinfrog/patchanalyzer:v0.0.1",
							ImagePullPolicy: "IfNotPresent",
							Command: []string{
								"sh",
								"-c",
								`
#!/bin/sh
chmod +x /vulas/patcheval.sh
sh /vulas/patcheval.sh
                `,
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      getChunkName(chunkID),
									MountPath: "/vulas/patcheval.sh",
									ReadOnly:  false,
								},
							},
							SecurityContext: &apiv1.SecurityContext{
								RunAsUser:              convert.Int64Ptr(0),
								ReadOnlyRootFilesystem: convert.BoolPtr(false),
							},
						},
					},
				},
			},
		},
	}

	log.Println(job)
	_, err := jobClient.Create(job)
	if err != nil {
		deleteErr := jobClient.Delete(job.Name, &metav1.DeleteOptions{})
		if deleteErr != nil {
			log.Fatal(err, deleteErr)
		}
		log.Fatal(err)
	}
}
