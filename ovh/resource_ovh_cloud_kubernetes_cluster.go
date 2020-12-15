package ovh

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ovh/go-ovh/ovh"
	"github.com/ovh/terraform-provider-ovh/ovh/helpers"
)

func resourceCloudProjectKube() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudProjectKubeCreate,
		Read:   resourceCloudProjectKubeRead,
		Delete: resourceCloudProjectKubeDelete,

		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				err := resourceCloudProjectKubeRead(d, meta)
				return []*schema.ResourceData{d}, err
			},
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OVH_PROJECT_ID", nil),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"control_plane_is_up_to_date": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"is_up_to_date": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"next_upgrade_versions": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"nodes_url": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					err := helpers.ValidateKubeRegion(v.(string))
					if err != nil {
						errors = append(errors, err)
					}
					return
				},
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"update_policy": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"url": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
			"client_certificate": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"client_key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"cluster_ca_certificate": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"kubeconfig": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"version": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					err := helpers.ValidateKubeVersion(v.(string))
					if err != nil {
						errors = append(errors, err)
					}
					return
				},
			},
		},
	}
}

type PublicCloudProjectKubeCreateOpts struct {
	Name    string `json:"name"`
	Region  string `json:"region"`
	Version string `json:"version"`
}

func resourceCloudProjectKubeCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	projectId := d.Get("project_id").(string)
	params := &PublicCloudProjectKubeCreateOpts{
		Name:    d.Get("name").(string),
		Region:  d.Get("region").(string),
		Version: d.Get("version").(string),
	}

	r := &PublicCloudProjectKubeResponse{}

	log.Printf("[DEBUG] Will create kubernetes cluster: %s", params)

	d.Partial(true)

	endpoint := fmt.Sprintf("/cloud/project/%s/kube", projectId)

	err := config.OVHClient.Post(endpoint, params, r)
	if err != nil {
		return fmt.Errorf("calling Post %s with params %s:\n\t %q", endpoint, params, err)
	}

	// This is a fix for a weird bug where the cluster is not immediately available on API
	log.Printf("[DEBUG] Waiting for cluster %s to be available", r.Id)

	bugFixWait := &resource.StateChangeConf{
		Pending:    []string{"NOT_FOUND"},
		Target:     []string{"FOUND"},
		Refresh:    waitForCloudProjectKubeToBeReal(config.OVHClient, projectId, r.Id),
		Timeout:    30 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = bugFixWait.WaitForState()
	if err != nil {
		return fmt.Errorf("timeout while creating cluster %s: %v", r.Id, err)
	}

	log.Printf("[DEBUG] Waiting for cluster %s to be READY", r.Id)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"INSTALLING"},
		Target:     []string{"READY"},
		Refresh:    waitForCloudProjectKubeActive(config.OVHClient, projectId, r.Id),
		Timeout:    20 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("timeout while waiting cluster %s to be READY: %v", r.Id, err)
	}
	log.Printf("[DEBUG] cluster %s is READY", r.Id)

	d.SetId(r.Id)
	err = resourceCloudProjectKubeRead(d, meta)

	if err != nil {
		return fmt.Errorf("error while reading cluster: %s", err)
	}

	d.Partial(false)

	return nil
}

func resourceCloudProjectKubeRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	projectId := d.Get("project_id").(string)

	d.Partial(true)
	r := &PublicCloudProjectKubeResponse{}

	log.Printf("[DEBUG] Will read cluster %s from project: %s", d.Id(), projectId)

	endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s", projectId, d.Id())

	err := config.OVHClient.Get(endpoint, r)
	if err != nil {
		return fmt.Errorf("calling Get %s:\n\t %q", endpoint, err)
	}

	err = readCloudProjectKube(projectId, config, d, r)
	if err != nil {
		return fmt.Errorf("error while reading cluster data %s:\n\t %q", d.Id(), err)
	}
	d.Partial(false)

	log.Printf("[DEBUG] Read cluster %+v", r)
	return nil
}

func resourceCloudProjectKubeDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	projectId := d.Get("project_id").(string)
	id := d.Id()

	log.Printf("[DEBUG] Will delete kubernetes cluster %s from project: %s", id, projectId)

	endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s", projectId, id)

	err := config.OVHClient.Delete(endpoint, nil)
	if err != nil {
		return fmt.Errorf("calling Delete %s:\n\t %q", endpoint, err)
	}

	log.Printf("[DEBUG] Waiting for cluster %s to be DELETED", id)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"DELETING"},
		Target:     []string{"DELETED"},
		Refresh:    waitForCloudProjectKubeDelete(config.OVHClient, projectId, id),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("timeout while waiting cluster %s to be deleted: %v", id, err)
	}

	d.SetId("")

	log.Printf("[DEBUG] cluster %s is DELETED", id)

	return nil
}

func cloudProjectKubeExists(projectId, id string, c *ovh.Client) error {
	r := &PublicCloudProjectKubeResponse{}

	log.Printf("[DEBUG] Will read kubernetes cluster for project: %s, id: %s", projectId, id)

	endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s", projectId, id)

	err := c.Get(endpoint, r)
	if err != nil {
		return fmt.Errorf("calling Get %s:\n\t %q", endpoint, err)
	}
	log.Printf("[DEBUG] Read cluster: %+v", r)

	return nil
}

func readCloudProjectKube(projectId string, config *Config, d *schema.ResourceData, cluster *PublicCloudProjectKubeResponse) (err error) {
	_ = d.Set("control_plane_is_up_to_date", cluster.ControlPlaneIsUpToDate)
	_ = d.Set("is_up_to_date", cluster.IsUpToDate)
	_ = d.Set("name", cluster.Name)
	_ = d.Set("next_upgrade_versions", cluster.NextUpgradeVersions)
	_ = d.Set("nodes_url", cluster.NodesUrl)
	_ = d.Set("region", cluster.Region)
	_ = d.Set("status", cluster.Status)
	_ = d.Set("update_policy", cluster.UpdatePolicy)
	_ = d.Set("url", cluster.Url)
	_ = d.Set("kubernetes_version", cluster.Version)

	if d.IsNewResource() {

		kubeconfigRaw := CloudProjectKubeKubeConfigResponse{}
		endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s/kubeconfig", projectId, cluster.Id)
		err = config.OVHClient.Post(endpoint, nil, &kubeconfigRaw)

		if err != nil {
			return err
		}
		_ = d.Set("kubeconfig", kubeconfigRaw.Content)

		kubeconfig, err := clientcmd.Load([]byte(kubeconfigRaw.Content))
		if err != nil {
			return err
		}
		currentContext := kubeconfig.CurrentContext
		currentUser := kubeconfig.Contexts[currentContext].AuthInfo
		currentCluster := kubeconfig.Contexts[currentContext].Cluster
		_ = d.Set("client_certificate", string(kubeconfig.AuthInfos[currentUser].ClientCertificateData))
		_ = d.Set("client_key", string(kubeconfig.AuthInfos[currentUser].ClientKeyData))
		_ = d.Set("cluster_ca_certificate", string(kubeconfig.Clusters[currentCluster].CertificateAuthorityData))

	}

	err = nil
	return
}

func waitForCloudProjectKubeActive(c *ovh.Client, projectId, cloudProjectKubeId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		r := &PublicCloudProjectKubeResponse{}
		endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s", projectId, cloudProjectKubeId)
		err := c.Get(endpoint, r)
		if err != nil {
			return r, "", err
		}

		return r, r.Status, nil
	}
}

func waitForCloudProjectKubeDelete(c *ovh.Client, projectId, CloudProjectKubeId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		r := &PublicCloudProjectKubeResponse{}
		endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s", projectId, CloudProjectKubeId)
		err := c.Get(endpoint, r)
		if err != nil {
			if err.(*ovh.APIError).Code == 404 {
				return r, "DELETED", nil
			} else {
				return r, "", err
			}
		}

		return r, r.Status, nil
	}
}

func waitForCloudProjectKubeToBeReal(client *ovh.Client, projectId string, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		r := &PublicCloudProjectKubeResponse{}
		endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s", projectId, id)
		err := client.Get(endpoint, r)
		if err != nil {
			if err.(*ovh.APIError).Code == 404 {
				return r, "NOT_FOUND", nil
			} else {
				return r, "", err
			}
		}

		return r, "FOUND", nil
	}
}
