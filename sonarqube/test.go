package main

import (
	"fmt"
	"log"
	"os"

	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

// Define a struct to represent the YAML structure
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

// LoadConfigFromFile loads YAML configuration from a file
func LoadConfigFromFile(filePath string) (*DeploymentConfig, error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	config := &DeploymentConfig{}
	err = yamlutil.Unmarshal(fileContent, config)
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

func UpdateNamespace(config *DeploymentConfig, newNamespace string) {
	config.Metadata.Namespace = newNamespace
}

// SaveConfigToFile saves the updated configuration to a file
func SaveConfigToFile(config *DeploymentConfig, filePath string) error {
	data, err := yamlutil.Marshal(config)
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
	// Load YAML configuration from file
	config, err := LoadConfigFromFile("dist/sonarqube1.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// Update the container image
	UpdateContainerImage(config, "sonarqube", "docker.io/sonarqube:developper")
	UpdateNamespace(config, "sonarqube1")

	// Save the updated configuration to a file
	err = SaveConfigToFile(config, "dist/sonarqube1.yaml")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Configuration updated successfully.")
}
