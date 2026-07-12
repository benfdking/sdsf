package provider

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var httpRequestReturnAttrTypes = map[string]attr.Type{
	"response_body": types.StringType,
	"status_code":   types.Int64Type,
	"timestamp":     types.StringType,
}

var _ function.Function = &HttpRequestFunction{}

type HttpRequestFunction struct{}

func isHttpRequestRetryMode() bool {
	return os.Getenv("HTTP_REQ_RETRY_MODE") != "false"
}

func NewFunctionHttpRequest() function.Function {
	return &HttpRequestFunction{}
}

func (f *HttpRequestFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "http_request"
}

func (f *HttpRequestFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Makes an HTTP request and returns the response body and status code",
		MarkdownDescription: `
Executes an HTTP request and returns the response body, status code, and the request timestamp.

**Environment variables to override parameters**

- "HTTP_REQ_RETRY_MODE": Enables/disables the "retryClient.RetryMax" mechanism, which is enabled by default.
`,

		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "url",
				MarkdownDescription: "URL to send the HTTP request. (e.g. https://google.com)",
			},
			function.StringParameter{
				Name:                "method",
				MarkdownDescription: "HTTP method (e.g. GET).",
			},
			function.StringParameter{
				Name:                "request_body",
				MarkdownDescription: "Request body to send with the HTTP request. (e.g. \"\" if not required)",
			},
			function.MapParameter{
				Name:                "headers",
				MarkdownDescription: "Headers for the HTTP request. Provide a map of key-value pairs representing header names and values.",
				ElementType:         types.StringType,
			},
		},
		Return: function.ObjectReturn{
			AttributeTypes: httpRequestReturnAttrTypes,
		},
	}
}

func (f *HttpRequestFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var inputs struct {
		URL         string            `tfsdk:"url"`
		Method      string            `tfsdk:"method"`
		RequestBody string            `tfsdk:"request_body"`
		Headers     map[string]string `tfsdk:"headers"`
	}

	resp.Error = function.ConcatFuncErrors(req.Arguments.Get(ctx, &inputs.URL, &inputs.Method, &inputs.RequestBody, &inputs.Headers))
	if resp.Error != nil {
		return
	}

	retryClient := retryablehttp.NewClient()

	if isHttpRequestRetryMode() {
		retryClient.CheckRetry = retryablehttp.DefaultRetryPolicy
	} else {
		retryClient.RetryMax = 0
	}

	httpReq, err := retryablehttp.NewRequest(inputs.Method, inputs.URL, []byte(inputs.RequestBody))
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, "Failed to create HTTP request")
		tflog.Error(ctx, "Failed to create HTTP request", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	for key, value := range inputs.Headers {
		httpReq.Header.Set(key, value)
	}

	httpResp, err := retryClient.Do(httpReq)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(2, "Failed to execute HTTP request")
		tflog.Error(ctx, "HTTP request failed", map[string]interface{}{
			"error": err.Error(),
		})
		httpResp = &http.Response{StatusCode: http.StatusInternalServerError}
	} else {
		defer httpResp.Body.Close()
	}

	bodyBytes := []byte(`{"error": "failed to retrieve response body"}`)
	if err == nil {
		bodyBytes, err = io.ReadAll(httpResp.Body)
		if err != nil {
			resp.Error = function.NewArgumentFuncError(3, "Failed to read HTTP response body")
			tflog.Error(ctx, "Failed to read response body", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	timestamp := time.Now().Format(time.RFC3339)
	httpResponseObj, diags := types.ObjectValue(
		httpRequestReturnAttrTypes,
		map[string]attr.Value{
			"response_body": types.StringValue(string(bodyBytes)),
			"status_code":   types.Int64Value(int64(httpResp.StatusCode)),
			"timestamp":     types.StringValue(timestamp),
		},
	)

	resp.Error = function.FuncErrorFromDiags(ctx, diags)
	if resp.Error != nil {
		return
	}

	resp.Error = resp.Result.Set(ctx, &httpResponseObj)
}
