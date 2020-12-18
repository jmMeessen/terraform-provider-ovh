package ovh

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func init() {
	resource.AddTestSweepers("ovh_cloud_project_kube", &resource.Sweeper{
		Name: "ovh_cloud_project_kube",
		F:    testSweepCloudProjectKube,
	})
}

func testSweepCloudProjectKube(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}

	serviceName := os.Getenv("OVH_CLOUD_PROJECT_KUBE_SERVICE_TEST")
	if serviceName == "" {
		log.Print("[DEBUG] OVH_CLOUD_PROJECT_KUBE_SERVICE_TEST is not set. No kube to sweep")
		return nil
	}
	kubeId := os.Getenv("OVH_CLOUD_PROJECT_KUBE_SERVICE_TEST")
	if kubeId == "" {
		log.Print("[DEBUG] OVH_CLOUD_PROJECT_KUBE_SERVICE_TEST is not set. No kube to sweep")
		return nil
	}

	pools := make([]CloudProjectKubeNodePoolResponse, 0)
	if err := client.Get(fmt.Sprintf("/cloud/project/%s/kube/%s/nodepool", serviceName, kubeId), &pools); err != nil {
		return fmt.Errorf("Error calling GET /cloud/project/%s/kube/%s/nodepool:\n\t %q", serviceName, kubeId, err)
	}

	if len(pools) == 0 {
		log.Print("[DEBUG] No pool to sweep")
		return nil
	}

	for _, p := range pools {
		if !strings.HasPrefix(p.Name, test_prefix) {
			continue
		}

		err = resource.Retry(5*time.Minute, func() *resource.RetryError {
			if err := client.Delete(fmt.Sprintf("/cloud/project/%s/kube/%s/nodepool/%s", serviceName, kubeId, p.Id), nil); err != nil {
				return resource.RetryableError(err)
			}
			// Successful delete
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

var testAccCloudProjectKubeConfig = fmt.Sprintf(`
resource "ovh_cloud_project_kube" "cluster" {
	service_name  = "%s"
	id = "%s"
	region = "GRA5"
	version = "1.18"
}
`, os.Getenv("OVH_CLOUD_PROJECT_SERVICE_TEST"), os.Getenv("OVH_CLOUD_PROJECT_KUBE_SERVICE_TEST"))

func TestAccCloudProjectKube_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckCloud(t)
			testAccCheckCloudProjectExists(t)
			testAccPreCheckKubernetes(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudProjectKubeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCloudProjectKubeConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudProjectKubeExists("ovh_cloud_project_kube.cluster", t),
				),
			},
		},
	})
}

func testAccCheckCloudProjectKubeExists(n string, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		if rs.Primary.Attributes["service_name"] == "" {
			return fmt.Errorf("No service name set")
		}

		return cloudProjectKubeExists(rs.Primary.Attributes["service_name"], rs.Primary.ID, config.OVHClient)
	}
}

func testAccCheckCloudProjectKubeDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ovh_cloud_project_kube" {
			continue
		}

		err := cloudProjectKubeExists(rs.Primary.Attributes["service_name"], rs.Primary.ID, config.OVHClient)
		if err == nil {
			return fmt.Errorf("cloud > Kubernetes Cluster still exists")
		}

	}
	return nil
}
