package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func Test_http_request(t *testing.T) {
	t.Setenv("HTTP_REQ_RETRY_MODE", "false")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK
		switch r.URL.Path {
		case "/404":
			status = http.StatusNotFound
		case "/500":
			status = http.StatusInternalServerError
		}
		w.WriteHeader(status)
	}))
	defer server.Close()

	for _, status := range []int{http.StatusOK, http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(fmt.Sprintf("status_%d", status), func(t *testing.T) {
			path := ""
			if status != http.StatusOK {
				path = fmt.Sprintf("/%d", status)
			}

			resource.UnitTest(t, resource.TestCase{
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{
					tfversion.SkipBelow(tfversion.Version1_8_0),
				},
				ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config: fmt.Sprintf(`
						output "test" {
							value = provider::slack::http_request(%q, "GET", "", {}).status_code
						}
					`, server.URL+path),
					Check: resource.TestCheckOutput("test", fmt.Sprintf("%d", status)),
				}},
			})
		})
	}
}
