package ovh

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/ovh/go-ovh/ovh"
)

func resourceCloudProjectKubeNodePool() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCloudProjectKubeNodePoolCreate,
		Read:          resourceCloudProjectKubeNodePoolRead,
		DeleteContext: resourceCloudProjectKubeNodePoolDelete,
		UpdateContext: resourceCloudProjectKubeNodePoolUpdate,

		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				err := resourceCloudProjectKubeNodePoolRead(d, meta)
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
			"cluster_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				Optional: false,
				ForceNew: true,
			},
			"flavor": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"desired_nodes": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: false,
			},
			"max_nodes": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: false,
			},
			"min_nodes": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: false,
			},
			"monthly_billed": {
				Type:     schema.TypeBool,
				Required: false,
				Optional: true,
				Default:  "false",

				ForceNew: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceCloudProjectKubeNodePoolUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	config := meta.(*Config)

	projectId := d.Get("project_id").(string)
	clusterId := d.Get("cluster_id").(string)
	poolId := d.Id()

	params := &CloudProjectKubeNodePoolUpdateRequest{
		DesiredNodes: d.Get("desired_nodes").(int),
		MaxNodes:     d.Get("max_nodes").(int),
		MinNodes:     d.Get("min_nodes").(int),
	}

	d.Partial(true)

	endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s/nodepool/%s", projectId, clusterId, poolId)

	log.Printf("[DEBUG] Will update nodepool: %+v", params)

	err := config.OVHClient.Put(endpoint, params, nil)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       "Failed to update nodepool",
			Detail:        err.Error(),
			AttributePath: nil,
		})
		return
	}

	log.Printf("[DEBUG] Waiting for nodepool %s to be READY", poolId)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"INSTALLING", "UPDATING", "REDEPLOYING", "RESIZING"},
		Target:     []string{"READY"},
		Refresh:    waitForCloudProjectKubeNodePoolActive(config.OVHClient, projectId, clusterId, poolId),
		Timeout:    45 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForStateContext(ctx)
	if err != nil {
		{
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "Timeout while waiting nodepool to be READY",
				Detail:        err.Error(),
				AttributePath: nil,
			})
			return
		}
	}

	log.Printf("[DEBUG] nodepool %s is READY", poolId)

	err = resourceCloudProjectKubeNodePoolRead(d, meta)

	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       "Failed to read nodepool",
			Detail:        err.Error(),
			AttributePath: nil,
		})
		return
	}

	d.Partial(false)

	return nil

}

func resourceCloudProjectKubeNodePoolCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	config := meta.(*Config)

	projectId := d.Get("project_id").(string)
	clusterId := d.Get("cluster_id").(string)

	params := &CloudProjectKubeNodePoolCreationRequest{
		Name:          d.Get("name").(string),
		FlavorName:    d.Get("flavor").(string),
		DesiredNodes:  d.Get("desired_nodes").(int),
		MaxNodes:      d.Get("max_nodes").(int),
		MinNodes:      d.Get("min_nodes").(int),
		MonthlyBilled: d.Get("monthly_billed").(bool),
	}

	r := &CloudProjectKubeNodePoolResponse{}

	d.Partial(true)

	endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s/nodepool", projectId, clusterId)

	log.Printf("[DEBUG] Will create nodepool: %+v", params)

	err := config.OVHClient.Post(endpoint, params, r)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       "Failed to create nodepool",
			Detail:        err.Error(),
			AttributePath: nil,
		})
		return
	}

	d.SetId(r.Id)

	// This is a fix for a weird bug where the nodepool is not immediately available on API
	log.Printf("[DEBUG] Waiting for nodepool %s to be available", r.Id)

	bugFixWait := &resource.StateChangeConf{
		Pending:    []string{"NOT_FOUND"},
		Target:     []string{"FOUND"},
		Refresh:    waitForCloudProjectKubeNodePoolToBeReal(config.OVHClient, projectId, clusterId, r.Id),
		Timeout:    2 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = bugFixWait.WaitForStateContext(ctx)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       "Timeout while creating nodepool",
			Detail:        err.Error(),
			AttributePath: nil,
		})
		return
	}

	log.Printf("[DEBUG] Waiting for nodepool %s to be READY", r.Id)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"INSTALLING", "UPDATING", "REDEPLOYING", "RESIZING"},
		Target:     []string{"READY"},
		Refresh:    waitForCloudProjectKubeNodePoolActive(config.OVHClient, projectId, clusterId, r.Id),
		Timeout:    10 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForStateContext(ctx)
	if err != nil {
		{
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "Timeout while waiting nodepool to be READY",
				Detail:        err.Error(),
				AttributePath: nil,
			})
			return
		}
	}

	log.Printf("[DEBUG] nodepool %s is READY", r.Id)

	d.SetId(r.Id)
	err = resourceCloudProjectKubeNodePoolRead(d, meta)

	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       "Failed to read nodepool",
			Detail:        err.Error(),
			AttributePath: nil,
		})
		return
	}

	d.Partial(false)

	return nil
}

func resourceCloudProjectKubeNodePoolRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	projectId := d.Get("project_id").(string)
	clusterId := d.Get("cluster_id").(string)

	r := &CloudProjectKubeNodePoolResponse{}

	d.Partial(true)

	endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s/nodepool/%s", projectId, clusterId, d.Id())

	log.Printf("[DEBUG] Will read nodepool %s from cluster %s in project %s", d.Id(), clusterId, projectId)

	err := config.OVHClient.Get(endpoint, r)
	if err != nil {
		return fmt.Errorf("fail to get nodepool on %s: %v", endpoint, err)
	}

	err = readCloudProjectKubeNodePool(projectId, config, d, r)
	if err != nil {
		return fmt.Errorf("error while reading nodepool data %s: %v", d.Id(), err)
	}
	d.Partial(false)

	log.Printf("[DEBUG] Read nodepool: %+v", r)
	return nil
}

func resourceCloudProjectKubeNodePoolDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	config := meta.(*Config)

	projectId := d.Get("project_id").(string)
	clusterId := d.Get("cluster_id").(string)
	id := d.Id()

	endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s/nodepool/%s", projectId, clusterId, id)

	log.Printf("[DEBUG] Will delete nodepool %s from cluster %s in project %s", id, clusterId, projectId)

	err := config.OVHClient.Delete(endpoint, nil)
	if err != nil {
		diags = append(diags, diag.Errorf("calling Delete %s:\n\t %q", endpoint, err)...)
		return
	}

	log.Printf("[DEBUG] Waiting for nodepool %s to be DELETED", id)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"DELETING"},
		Target:     []string{"DELETED"},
		Refresh:    waitForCloudProjectKubeNodePoolDelete(config.OVHClient, projectId, clusterId, id),
		Timeout:    45 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForStateContext(ctx)
	if err != nil {
		diags = append(diags, diag.Errorf("timeout while deleting nodepool %s from project %s", id, projectId)...)
		return
	}

	d.SetId("")

	log.Printf("[DEBUG] nodepool %s is DELETED", id)

	return
}

func cloudProjectKubeNodePoolExists(projectId string, clusterId string, id string, c *ovh.Client) error {
	r := &CloudProjectKubeNodePoolResponse{}

	endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s/nodepool/%s", projectId, clusterId, id)

	log.Printf("[DEBUG] Will check if nodepool %s from cluster %s in project %s exist", id, clusterId, projectId)

	err := c.Get(endpoint, r)
	if err != nil {
		return fmt.Errorf("fail to get nodepool on %s: %v", endpoint, err)
	}
	log.Printf("[DEBUG] Read nodepool: %+v", r)

	return nil
}

func readCloudProjectKubeNodePool(projectId string, config *Config, d *schema.ResourceData, cluster *CloudProjectKubeNodePoolResponse) (err error) {
	_ = d.Set("name", cluster.Name)
	_ = d.Set("flavor", cluster.Flavor)
	_ = d.Set("desired_nodes", cluster.DesiredNodes)
	_ = d.Set("max_nodes", cluster.MaxNodes)
	_ = d.Set("min_nodes", cluster.MinNodes)
	_ = d.Set("monthly_billed", cluster.MonthlyBilled)
	_ = d.Set("status", cluster.Status)
	d.SetId(cluster.Id)
	err = nil
	return
}

func waitForCloudProjectKubeNodePoolActive(c *ovh.Client, projectId, clusterId string, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		r := &CloudProjectKubeNodeResponse{}
		endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s/nodepool/%s", projectId, clusterId, id)
		err := c.Get(endpoint, r)
		if err != nil {
			return r, "", err
		}

		return r, r.Status, nil
	}
}

func waitForCloudProjectKubeNodePoolDelete(c *ovh.Client, projectId, clusterId string, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		r := &CloudProjectKubeNodeResponse{}
		endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s/nodepool/%s", projectId, clusterId, id)
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

func waitForCloudProjectKubeNodePoolToBeReal(client *ovh.Client, projectId string, clusterId string, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		r := &CloudProjectKubeNodeResponse{}
		endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s/nodepool/%s", projectId, clusterId, id)
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
