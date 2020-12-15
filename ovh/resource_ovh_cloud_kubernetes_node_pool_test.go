package ovh

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var testAccPublicCloudProjectKubeNodePoolConfig = fmt.Sprintf(`
data "ovh_cloud_project_kube" "cluster" {
	project_id  = "%s"
	name = "%s"
}
resource "ovh_cloud_project_kube_node_pool" "pool" {
	project_id  = "%s"
	cluster_id = data.ovh_cloud_project_kube.cluster.id
  	name = "acceptance-node"
	flavor = "b2-7"
	desired_size = 1
	min_size = 0
	max_size = 1
}
`, os.Getenv("OVH_PUBLIC_CLOUD"), os.Getenv("OVH_KUBERNETES_CLUSTER_NAME"), os.Getenv("OVH_PUBLIC_CLOUD"))

func TestAccPublicCloudProjectKubeNodePool_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckCloud(t)
			testAccCheckCloudProjectExists(t)
			testAccPreCheckKubernetes(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPublicCloudProjectKubeNodePoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPublicCloudProjectKubeNodePoolConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPublicCloudProjectKubeNodePoolExists("ovh_cloud_project_kube_node_pool.pool", t),
				),
			},
		},
	})
}

func testAccCheckPublicCloudProjectKubeNodePoolExists(n string, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		if rs.Primary.Attributes["project_id"] == "" {
			return fmt.Errorf("no Project ID is set")
		}

		return cloudProjectKubeNodePoolExists(rs.Primary.Attributes["project_id"], rs.Primary.Attributes["cluster_id"], rs.Primary.ID, config.OVHClient)
	}
}

func testAccCheckPublicCloudProjectKubeNodePoolDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ovh_cloud_project_kube" {
			continue
		}

		err := cloudProjectKubeNodePoolExists(rs.Primary.Attributes["project_id"], rs.Primary.Attributes["cluster_id"], rs.Primary.ID, config.OVHClient)
		if err == nil {
			return fmt.Errorf("cloud > Kubernetes Pool still exists")
		}

	}
	return nil
}
