package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/jsii-runtime-go"
	"github.com/briandowns/spinner"
	"github.com/golang/glog"
	_ "github.com/lib/pq"
	yaml1 "gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

type Configuration struct {
	ClusterName    string
	NSDataBase     string
	PvcDBsize      string
	PGSecret       string
	NSSonar        string
	PvcSonar       string
	StorageClass   string
	Sonaruser      string
	Sonarpass      string
	PGsql          string
	PGconf         string
	DepSonar       string
	PGsvc          string
	SonarSVC       string
	SonarPort      string
	SonarTransport string
	SonarTagImage  string
}

type ConfAuth struct {
	Region     string
	Account    string
	SSOProfile string
	Index      string
	AWSsecret  string
}

type Token struct {
	Login          string `json:"login"`
	Name           string `json:"name"`
	CreatedAt      string `json:"createdAt"`
	ExpirationDate string `json:"expirationDate"`
	Token          string `json:"token"`
	Type           string `json:"type"`
}

// Define the YAML structure for sonarqube manifest
type DeploymentConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec struct {
		Replicas int `yaml:"replicas"`
		Selector struct {
			MatchLabels struct {
				App string `yaml:"app"`
			} `yaml:"matchLabels"`
		} `yaml:"selector"`
		Template struct {
			Metadata struct {
				Name   string `yaml:"name"`
				Labels struct {
					App string `yaml:"app"`
				} `yaml:"labels"`
			} `yaml:"metadata"`
			Spec struct {
				SecurityContext struct {
					RunAsUser  int `yaml:"runAsUser"`
					RunAsGroup int `yaml:"runAsGroup"`
				} `yaml:"securityContext"`
				InitContainers []struct {
					Name            string   `yaml:"name"`
					Image           string   `yaml:"image"`
					Command         []string `yaml:"command"`
					ImagePullPolicy string   `yaml:"imagePullPolicy"`
					SecurityContext struct {
						Privileged bool `yaml:"privileged"`
						RunAsUser  int  `yaml:"runAsUser"`
					} `yaml:"securityContext"`
					VolumeMounts []struct {
						MountPath string `yaml:"mountPath"`
						Name      string `yaml:"name"`
					} `yaml:"volumeMounts"`
				} `yaml:"initContainers"`
				Containers []struct {
					Name            string `yaml:"name"`
					Image           string `yaml:"image"`
					ImagePullPolicy string `yaml:"imagePullPolicy"`
					Ports           []struct {
						ContainerPort int `yaml:"containerPort"`
					} `yaml:"ports"`
					VolumeMounts []struct {
						MountPath string `yaml:"mountPath"`
						Name      string `yaml:"name"`
					} `yaml:"volumeMounts"`
					Env []struct {
						Name      string `yaml:"name"`
						ValueFrom struct {
							SecretKeyRef struct {
								Name string `yaml:"name"`
								Key  string `yaml:"key"`
							} `yaml:"secretKeyRef"`
						} `yaml:"valueFrom"`
					} `yaml:"env"`
					SecurityContext struct {
						Privileged bool `yaml:"privileged"`
						RunAsUser  int  `yaml:"runAsUser"`
						RunAsGroup int  `yaml:"runAsGroup"`
					} `yaml:"securityContext"`
				} `yaml:"containers"`
				Volumes []struct {
					Name                  string `yaml:"name"`
					PersistentVolumeClaim struct {
						ClaimName string `yaml:"claimName"`
					} `yaml:"persistentVolumeClaim"`
				} `yaml:"volumes"`
			} `yaml:"spec"`
		} `yaml:"template"`
	} `yaml:"spec"`
}

func GetConfig(configcrd ConfAuth, configjs Configuration) (ConfAuth, Configuration) {

	fconfig, err := os.ReadFile("config.json")
	if err != nil {
		panic("❌ Problem with the configuration file : config.json")
		os.Exit(1)
	}
	if err := json.Unmarshal(fconfig, &configjs); err != nil {
		fmt.Println("❌ Error unmarshaling JSON:", err)
		os.Exit(1)
	}

	fconfig2, err := os.ReadFile("../config_crd.json")
	if err != nil {
		panic("❌ Problem with the configuration file : config_crd.json")
		os.Exit(1)
	}
	if err := json.Unmarshal(fconfig2, &configcrd); err != nil {
		fmt.Println("❌ Error unmarshaling JSON:", err)
		os.Exit(1)
	}
	return configcrd, configjs
}

func openAWSSession(region string) *secretsmanager.SecretsManager {
	// Open AWS session
	os.Setenv("AWS_SDK_LOAD_CONFIG", "true")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})

	if err != nil {
		log.Fatalf("❌ Error creating AWS session: %v", err)
		os.Exit(1)
	}

	// Create an AWS Secrets Manager service client
	svc := secretsmanager.New(sess)

	return svc
}

func applyResourcesFromYAML(yamlContent []byte, clientset *kubernetes.Clientset, dd *dynamic.DynamicClient, ns string) error {
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(yamlContent), 100)

	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		if err != nil {
			return err
		}
		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return err
		}
		unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}
		gr, err := restmapper.GetAPIGroupResources(clientset.Discovery())
		if err != nil {
			return err
		}
		mapper := restmapper.NewDiscoveryRESTMapper(gr)
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}
		var dri dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace(ns)
			}
			dri = dd.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dri = dd.Resource(mapping.Resource)
		}
		_, err = dri.Create(context.Background(), unstructuredObj, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func waitForServiceReady(clientset *kubernetes.Clientset, serviceName, namespace string, pollingInterval time.Duration) (string, string, error) {
	for {
		service, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
		if err != nil {
			return "", "", err
		}

		if len(service.Status.LoadBalancer.Ingress) > 0 {
			externalIP := service.Status.LoadBalancer.Ingress[0].Hostname
			if externalIP != "" {
				clusterIP := service.Spec.ClusterIP
				return externalIP, clusterIP, nil
			}
		}

		time.Sleep(pollingInterval)
	}
}

func deleteNamespace(clientset *kubernetes.Clientset, namespace string) error {
	// Delete the namespace.
	err := clientset.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	// Wait for the namespace to be deleted.
	for {
		_, err := clientset.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				fmt.Printf("\n✅ Namespace %s has been deleted\n", namespace)
				break
			}
		}
		time.Sleep(2 * time.Second)
	}

	return nil
}

func waitForDNSResolution(dnsName string) {
	spin1 := spinner.New(spinner.CharSets[37], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
	spin1.Prefix = " Waiting DNS resolution for Database service..."
	spin1.Start()
	for {
		_, err := net.LookupIP(dnsName)
		if err == nil {
			spin1.Stop()
			fmt.Printf("\n✅ DNS resolution for Database service is successful.\n")
			break
		}

		time.Sleep(3 * time.Second)
	}
}

// LoadConfigFromFile loads YAML configuration from a file
func LoadConfigFromFile(filePath string) (*DeploymentConfig, error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	config := &DeploymentConfig{}
	err = yaml1.Unmarshal(fileContent, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// UpdateContainerImage updates the image field of the specified container
func UpdateContainerImage(config *DeploymentConfig, containerName, newImage string) {
	for i := range config.Spec.Template.Spec.Containers {
		if config.Spec.Template.Spec.Containers[i].Name == containerName {
			config.Spec.Template.Spec.Containers[i].Image = newImage
			break
		}
	}
}

// SaveConfigToFile saves the updated configuration to a file
func SaveConfigToFile(config *DeploymentConfig, filePath string) error {
	data, err := yaml1.Marshal(config)
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, data, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func main() {

	var configcrd ConfAuth
	var config1 Configuration
	var AppConfig1, AppConfig = GetConfig(configcrd, config1)

	fmt.Printf("Test ID :", AppConfig1.Account)

	pollingInterval := 5 * time.Second

	configMapData := make(map[string]string, 0)
	initdb := `
	psql -v ON_ERROR_STOP=1 --username "postgres" --dbname "postgres" <<-EOSQL
	CREATE ROLE ` + AppConfig.Sonaruser + ` WITH LOGIN PASSWORD '` + AppConfig.Sonarpass + `';
	CREATE DATABASE sonarqube WITH ENCODING 'UTF8' OWNER ` + AppConfig.Sonaruser + ` TEMPLATE=template0;
	GRANT ALL PRIVILEGES ON DATABASE sonarqube TO ` + AppConfig.Sonaruser + `;
	EOSQL
	`
	configMapData["init.sh"] = initdb

	// Parse command-line arguments
	cmdArgs := os.Args[1:]

	// Load Kubeconfig
	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("❌ Failed to create a ClientSet: %v. Exiting.", err)
	}

	if len(cmdArgs) != 1 || (cmdArgs[0] != "deploy" && cmdArgs[0] != "destroy") {
		fmt.Println("❌ Usage: go run main.go [deploy|destroy]")
		os.Exit(1)
	}

	/*------------------------- Main -----------------------------*/

	if cmdArgs[0] == "deploy" {

		spin := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		spin.Prefix = "Deployment PostgreSQL Database : "
		spin.Color("green", "bold")
		spin.Start()

		fmt.Printf("\r%s %s \n", spin.Prefix, "Creating namespace...")
		// Create a Namespace Database
		nsName := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: AppConfig.NSDataBase,
			},
		}
		_, err = clientset.CoreV1().Namespaces().Create(context.Background(), nsName, metav1.CreateOptions{})
		if err != nil {
			spin.Stop()
			fmt.Printf("❌ Error creating namespace: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\r✅ Namespace %s created successfully\n", AppConfig.NSDataBase)

		fmt.Printf("\r%s %s \n", spin.Prefix, "Creating PVC...")

		// Create a PVC for database
		pvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pgsql-data",
				Namespace: AppConfig.NSDataBase,
			},
			Spec: v1.PersistentVolumeClaimSpec{
				AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				StorageClassName: &AppConfig.StorageClass,
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceStorage: resource.MustParse(AppConfig.PvcDBsize),
					},
				},
			},
		}
		_, err := clientset.CoreV1().PersistentVolumeClaims(AppConfig.NSDataBase).Create(context.TODO(), pvc, metav1.CreateOptions{})
		if err != nil {
			spin.Stop()
			fmt.Printf("\n❌ Error creating PVC: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\r✅ PVC Database : pgsql-data created successfully\n")

		fmt.Printf("\r%s %s \n", spin.Prefix, "Creating secret database...")

		//Create a secret database
		dd, err := dynamic.NewForConfig(config)
		if err != nil {
			log.Fatal(err)
		}

		pvYAML, err := os.ReadFile(AppConfig.PGSecret)
		if err != nil {
			spin.Stop()
			fmt.Printf("\n ❌ Error reading Secret YAML file %s: %v\n", err, AppConfig.PGSecret)
			os.Exit(1)
		}
		err = applyResourcesFromYAML(pvYAML, clientset, dd, AppConfig.NSDataBase)
		if err != nil {
			spin.Stop()
			log.Fatalf("\n ❌ Error applying %s file %v\n", err, AppConfig.PGSecret)
			return
		}
		fmt.Println("\r✅ Database secret created successfully\n")

		fmt.Printf("\r%s %s \n", spin.Prefix, "Creating ConfigMap Init DB...")

		// Create a ConfigMap Init DB
		PGsqlInit := v1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pgsql-init",
				Namespace: AppConfig.NSDataBase,
			},
			Data: configMapData,
		}
		_, err1 := clientset.CoreV1().ConfigMaps(AppConfig.NSDataBase).Create(context.TODO(), &PGsqlInit, metav1.CreateOptions{})
		if err1 != nil {
			spin.Stop()
			fmt.Printf("\n ❌ Error creating PGSQLInit configMaps: %v\n", err1)
			os.Exit(1)
		}
		fmt.Println("\r✅ PGSQLInit configMaps created successfully\n")

		fmt.Printf("\r%s %s \n", spin.Prefix, "Creating ConfigMap DATA DB...")
		// Create a ConfigMap DATA DB
		pgcYAML, err := os.ReadFile(AppConfig.PGconf)
		if err != nil {
			spin.Stop()
			fmt.Printf("\n ❌ Error reading Secret YAML file %s: %v\n", err, AppConfig.PGconf)
			os.Exit(1)
		}
		err = applyResourcesFromYAML(pgcYAML, clientset, dd, AppConfig.NSDataBase)
		if err != nil {
			spin.Stop()
			log.Fatalf("\n ❌ Error applying %s file %v\n", err, AppConfig.PGconf)
			return
		}
		fmt.Println("\r✅ PGSQLData configMaps created successfully\n")

		fmt.Printf("\r%s %s \n", spin.Prefix, "Deploy Postgresql deployment...")

		// Deploy Postgresql

		pgYAML, err := os.ReadFile(AppConfig.PGsql)
		if err != nil {
			spin.Stop()
			fmt.Printf("\n ❌ Error reading PGSQL YAML file %s: %v\n", err, AppConfig.PGsql)
			os.Exit(1)
		}
		err = applyResourcesFromYAML(pgYAML, clientset, dd, AppConfig.NSDataBase)
		if err != nil {
			spin.Stop()
			log.Fatalf("\n ❌ Error applying %s file %v\n", err, AppConfig.PGsql)
			return
		}

		externalIP, ClusterIP, err := waitForServiceReady(clientset, AppConfig.PGsvc, AppConfig.NSDataBase, pollingInterval)
		if err != nil {
			spin.Stop()
			fmt.Printf("\n ❌ Error waiting for service to become ready: %v\n", err)
			os.Exit(1)
		}
		JDBCURL := "jdbc:postgresql://" + AppConfig.PGsvc + "." + AppConfig.NSDataBase + ".svc.cluster.local:5432/sonarqube?currentSchema=public"
		spin.Stop()
		fmt.Printf("\n✅ PostgreSQL Database Successful deployment External IP: %s\n", externalIP)
		fmt.Printf("✅ JDBC URL : %s - IP : %s\n\n\n", JDBCURL, ClusterIP)

		spin.Prefix = "Deployment SonarQube : "
		spin.Start()

		fmt.Printf("\r%s %s \n", spin.Prefix, "Creating namespace...")
		// Create Namespace sonarqube

		nsNameS := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: AppConfig.NSSonar,
			},
		}
		_, err = clientset.CoreV1().Namespaces().Create(context.Background(), nsNameS, metav1.CreateOptions{})
		if err != nil {
			spin.Stop()
			fmt.Printf("\n ❌ Error creating namespace: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\r✅ Namespace %s created successfully\n", AppConfig.NSSonar)

		fmt.Printf("\r%s %s \n", spin.Prefix, "creating PVCs...")

		// Create PVCs for sonarqube

		sonarYAML, err := os.ReadFile(AppConfig.PvcSonar)
		if err != nil {
			spin.Stop()
			fmt.Printf("\n ❌ Error reading PVCSONAR YAML file %s: %v\n", err, AppConfig.PvcSonar)
			os.Exit(1)
		}
		err = applyResourcesFromYAML(sonarYAML, clientset, dd, AppConfig.NSSonar)
		if err != nil {
			spin.Stop()
			log.Fatalf("\n❌ Error applying %s file %v\n", err, AppConfig.PvcSonar)
			os.Exit(1)
		}

		fmt.Printf("\r✅ SonarQube PVCs created successfully\n")

		fmt.Printf("\r%s %s \n", spin.Prefix, "Creating sonar k8s secret...")
		// Define the data for the Secret.
		secretData := map[string][]byte{
			"SONAR_JDBC_USERNAME": []byte(AppConfig.Sonaruser),
			"SONAR_JDBC_PASSWORD": []byte(AppConfig.Sonarpass),
			"SONAR_JDBC_URL":      []byte(JDBCURL),
		}

		// Create the Secret object.
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sonarsecret",
				Namespace: AppConfig.NSSonar,
			},
			Data: secretData,
			Type: v1.SecretTypeOpaque,
		}

		// Create the Secret in the cluster.
		createdSecret, err := clientset.CoreV1().Secrets(AppConfig.NSSonar).Create(context.Background(), secret, metav1.CreateOptions{})
		if err != nil {
			spin.Stop()
			fmt.Printf("\n❌ Error creating Secret: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\r✅ SonarQube k8s Secret for Database created successfully : %s\n", createdSecret.Name)

		// Modify Sonarqube manifest : namespace and image tag : community, developer, enterprise
		// show different image tag : https://hub.docker.com/_/sonarqube/tags

		fmt.Printf("\r%s %s \n", spin.Prefix, "Updating sonarqube image tag...")
		// Load YAML configuration from file
		config, err := LoadConfigFromFile(AppConfig.DepSonar)
		if err != nil {
			log.Fatal(err)
		}

		// Update the container image
		UpdateContainerImage(config, "sonarqube", AppConfig.SonarTagImage)

		// Save the updated configuration to a file
		err = SaveConfigToFile(config, AppConfig.DepSonar)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("\r✅ Manifest sonarqube.yml updated successfully")
		spin.Stop()

		fmt.Printf("\r%s %s \n", spin.Prefix, "Deployment SonarQube POD...")
		// Deploy SonarQube pods

		sonardYAML, err := os.ReadFile(AppConfig.DepSonar)
		if err != nil {
			spin.Stop()
			fmt.Printf("\n❌ Error reading SONARQUBE YAML file %s: %v\n", err, AppConfig.DepSonar)
			os.Exit(1)
		}
		err = applyResourcesFromYAML(sonardYAML, clientset, dd, AppConfig.NSSonar)
		if err != nil {
			spin.Stop()
			log.Fatalf("\n❌ Error applying %s file %v\n", err, AppConfig.DepSonar)
			os.Exit(1)
		}
		fmt.Printf("\r✅ SonarQube Pod Successful deployment\n")

		fmt.Printf("\r%s %s \n", spin.Prefix, "Deployment SonarQube Service...")
		// Deploy SonarQube Service

		sonarsvcYAML, err := os.ReadFile("dist/sonarsvc.yaml")
		if err != nil {
			spin.Stop()
			fmt.Printf("\n❌ Error reading SONARQUBE Service YAML file sonarsvc.yaml : %v\n", err)
			os.Exit(1)
		}
		err = applyResourcesFromYAML(sonarsvcYAML, clientset, dd, AppConfig.NSSonar)
		if err != nil {
			spin.Stop()
			log.Fatalf("\n❌ Error applying sonarsvc.yaml file %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\r✅ SonarQube Service Successful deployment\n")

		fmt.Printf("\r%s %s \n", spin.Prefix, "Waiting SonarQube Service up...")

		externalIPS, _, err := waitForServiceReady(clientset, AppConfig.SonarSVC, AppConfig.NSSonar, pollingInterval)
		if err != nil {
			spin.Stop()
			fmt.Printf("\n❌ Error waiting for service to become ready: %v\n", err)
			os.Exit(1)
		}
		SONARURL := AppConfig.SonarTransport + externalIPS + ":9000"

		fmt.Printf("\n\n✅ SonarQube deployment created successfully - External Connexion: %s\n", SONARURL)
		fmt.Printf("\r✅ SonarQube deployment created successfully 😀\n\n")
		spin.Stop()

		waitForDNSResolution(externalIP)

		/*--------------------------------------- Set SonarQube License -------------------------------*/
		/* This part is Optionnal, it applies the license file for sonarqube (the license.lic file must be located in the directory where the deployment is launched) */

		/*	spin.Start()

			// Create a PostgreSQL connection string
			connectionString1 := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
				externalIP, "5432", "sonarqube", AppConfig.Sonaruser, AppConfig.Sonarpass)

			// Connect to the PostgreSQL database
			db, err := sql.Open("postgres", connectionString1)
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()

			// Get timestamp
			currentTime := time.Now()
			TimestampMillis := currentTime.UnixNano() / int64(time.Millisecond)

			Query3 := "INSERT INTO public.internal_properties(kee, is_empty, text_value, clob_value, created_at) VALUES ('sonarsource.license',false, $1, '', $2);"

			// Read the content of the text file
			filePath := "license.lic"
			content, err := os.ReadFile(filePath)
			if err != nil {
				log.Fatal(err)
			}

			// Convert the content to a string
			fileContent := string(content)

			_, err = db.Exec(Query3, fileContent, TimestampMillis)
			if err != nil {
				spin.Stop()
				log.Fatal(err)
			} else {
				spin.Stop()
				fmt.Println("✅ UPDATE Lisence executed successfully.")
			}*/

		/*------------------------------Generated SonarQube Token and store in AWS secret ----------------------*/

		SonarHostURL := AppConfig.SonarTransport + externalIPS + ":" + AppConfig.SonarPort
		spin.Prefix = "Generated SonarQube Token :"
		spin.Start()
		baseURL := SonarHostURL + "/api/user_tokens/generate"
		payload := []byte(`{}`)

		username := "admin"
		password := "admin"

		fmt.Printf("\r%s %s \n", spin.Prefix, "Creating Token...")

		// Create a URL with query parameters
		u, err := url.Parse(baseURL)
		if err != nil {
			spin.Stop()
			fmt.Println("❌ Error parsing URL:", err)
			os.Exit(1)
		}

		// Add query parameters to the URL
		q := u.Query()
		q.Set("name", "awsanalyse")
		q.Set("type", "GLOBAL_ANALYSIS_TOKEN")
		u.RawQuery = q.Encode()

		req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(payload))
		if err != nil {
			spin.Stop()
			fmt.Println("❌ Error creating request:", err)
			os.Exit(1)
		}

		// Set the Content-Type header
		req.Header.Set("Content-Type", "application/json")

		// Set basic authentication
		req.SetBasicAuth(username, password)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			spin.Stop()
			fmt.Println("❌ Error sending request:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			spin.Stop()
			fmt.Printf("❌ Request failed with status code: %d\n", resp.StatusCode)
			os.Exit(1)
		}

		var response Token
		decoder := json.NewDecoder(resp.Body)
		if err := decoder.Decode(&response); err != nil {
			fmt.Println("❌ Error decoding JSON response:", err)
			os.Exit(1)
		}

		fmt.Printf("\r✅ Token creation successful : SONAR_TOKEN= %s\n", response.Token)

		/*----------------------------- Add Token in AWS Secret ------------------------------*/

		fmt.Printf("\r%s %s \n", spin.Prefix, "Add Token in AWS Secret...")

		// Open AWS session
		// Create a AWS Secrets Manager service client
		svc := openAWSSession(AppConfig1.Region)

		// Define the secret name and the secret key-value pairs
		secretName := AppConfig1.AWSsecret + AppConfig1.Index

		secretData02 := map[string]string{
			"SONAR_JDBC_USERNAME": AppConfig.Sonaruser,
			"SONAR_JDBC_PASSWORD": AppConfig.Sonarpass,
			"SONAR_JDBC_URL":      JDBCURL,
			"SONAR_HOST_URL":      SonarHostURL,
			"SONAR_TOKEN":         response.Token,
		}

		// Convert the secret data to JSON format
		jsonData := "{"

		for key, value := range secretData02 {
			jsonData += `"` + key + `": "` + value + `",`
		}

		jsonData = jsonData[:len(jsonData)-1] + "}"

		// Create the AWS secret

		_, err = svc.CreateSecret(&secretsmanager.CreateSecretInput{
			Name:         &secretName,
			SecretString: &jsonData,
			Description:  jsii.String("AWS Workshop SonarQube Database Connexion"),
		})

		if err != nil {
			spin.Stop()
			fmt.Println("\n ❌ Error creating secret:", err)
			os.Exit(1)
		}
		fmt.Println("\r✅ AWS Secret created successfully:", secretName)
		spin.Stop()

	} else if cmdArgs[0] == "destroy" {

		/*--------------------------------- Destroy Steps ------------------------------------*/

		// Open AWS Session
		svc := openAWSSession(AppConfig1.Region)

		spin := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		spin.Prefix = "Destroy Deployment SonarQube : "
		spin.Color("green", "bold")
		spin.Start()

		fmt.Printf("\r%s %s \n", spin.Prefix, "Destroy the SonarQube Namespace...")
		// Destroy the SonarQube Namespace
		if err := deleteNamespace(clientset, AppConfig.NSSonar); err != nil {
			spin.Stop()
			fmt.Printf("\n❌ Error deleting namespace %s: %v\n", AppConfig.NSSonar, err)
			os.Exit(1)
		}
		fmt.Println("\r✅ Deployment SonarQube deleted successfully\n")
		spin.Stop()

		spin.Suffix = "Destroy Deployment Database : "
		spin.Start()

		fmt.Printf("\r%s %s \n", spin.Prefix, "Destroy the Database Namespace...")
		// Destroy the Database Namespace
		if err := deleteNamespace(clientset, AppConfig.NSDataBase); err != nil {
			spin.Stop()
			fmt.Printf("\n❌ Error deleting namespace %s: %v\n", AppConfig.NSDataBase, err)
			os.Exit(1)
		}
		fmt.Println("\n ✅ Deployment Database deleted successfully\n")
		spin.Stop()

		spin.Prefix = "Destroy AWS Secret ..."
		spin.Start()
		// Delete Secret
		secretName := AppConfig1.AWSsecret + AppConfig1.Index

		// Create the input for deleting the secret
		input := &secretsmanager.DeleteSecretInput{
			SecretId:                   &secretName,
			ForceDeleteWithoutRecovery: aws.Bool(true),
		}
		// Delete the secret
		_, err = svc.DeleteSecret(input)

		if err != nil {
			spin.Stop()
			fmt.Println("\n❌ Error deleting secret:", err)
			os.Exit(1)
		}
		fmt.Println("\n ✅ Secret deleted successfully:", secretName)
		spin.Stop()

	}

}
