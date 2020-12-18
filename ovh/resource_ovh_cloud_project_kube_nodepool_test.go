package ovh

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var testAccCloudProjectKubeNodePoolConfig = fmt.Sprintf(`
data "ovh_cloud_project_kube" "cluster" {
	service_name  = "%s"
	id = "%s"
}
resource "ovh_cloud_project_kube_nodepool" "pool" {
	service_name  = "%s"
	cluster_id = data.ovh_cloud_project_kube.cluster.id
		name = "%s"
	flavor = "b2-7"
	desired_size = 1
	min_size = 0
	max_size = 1
}
`, os.Getenv("OVH_CLOUD_PROJECT_SERVICE_TEST"), os.Getenv("OVH_CLOUD_PROJECT_KUBE_SERVICE_TEST"),
	os.Getenv("OVH_CLOUD_PROJECT_SERVICE_TEST"), acctest.RandomWithPrefix(test_prefix))

func TestAccCloudProjectKubeNodePool_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckCloud(t)
			testAccCheckCloudProjectExists(t)
			testAccPreCheckKubernetes(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudProjectKubeNodePoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCloudProjectKubeNodePoolConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudProjectKubeNodePoolExists("ovh_cloud_project_kube_nodepool.pool", t),
				),
			},
		},
	})
}

func testAccCheckCloudProjectKubeNodePoolExists(n string, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		if rs.Primary.Attributes["service_name"] == "" {
			return fmt.Errorf("no service name set")
		}

		return cloudProjectKubeNodePoolExists(rs.Primary.Attributes["service_name"], rs.Primary.Attributes["cluster_id"], rs.Primary.ID, config.OVHClient)
	}
}

func testAccCheckCloudProjectKubeNodePoolDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ovh_cloud_project_kube" {
			continue
		}

		err := cloudProjectKubeNodePoolExists(rs.Primary.Attributes["service_name"], rs.Primary.Attributes["cluster_id"], rs.Primary.ID, config.OVHClient)
		if err == nil {
			return fmt.Errorf("cloud > Kubernetes Pool still exists")
		}

	}
	return nil
}
