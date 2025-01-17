package wiremock

import (
	"encoding/json"
	"fmt"
	"github.com/prongbang/wiremock/v2/pkg/config"
	"github.com/prongbang/wiremock/v2/pkg/core"
	"github.com/prongbang/wiremock/v2/pkg/status"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
)

type UseCase interface {
	CasesMatching(r *http.Request, path string, cases map[string]Cases, params Parameters) CaseMatching
	ParameterMatching(params Parameters) Matching
	GetMockResponse(resp Response) []byte
	ReadSourceRouteYml(routeName string) []byte
	GetRoutes(filepath string) Routes
}

type useCase struct {
}

func (u *useCase) GetRoutes(filepath string) Routes {
	// Read yaml config
	source := u.ReadSourceRouteYml(filepath)

	// Unmarshal yaml config
	routes := Routes{}
	err := yaml.Unmarshal(source, &routes)
	if err != nil {
		panic(err)
	}
	return routes
}

func (u *useCase) CasesMatching(r *http.Request, path string, cases map[string]Cases, params Parameters) CaseMatching {

	// Get request
	body := core.Body(r)

	// Process header matching
	require := map[string]interface{}{}
	errors := map[string]interface{}{}
	matchingHeader := 0
	for k, v := range params.ReqHeader.MockHeader {
		vs := fmt.Sprintf("%v", v)
		ks := fmt.Sprintf("%v", params.ReqHeader.HttpHeader[k])
		if vs == ks {
			matchingHeader = matchingHeader + 1
			continue
		}
		if params.ReqHeader.HttpHeader[k] == nil {
			errors[k] = "Require header " + k
		} else {
			errors[k] = "The header " + k + " not match"
		}
	}
	if len(errors) > 0 {
		require["errors"] = errors
	}
	require["message"] = "validation error"
	require["status"] = "error"
	result, err := json.Marshal(require)
	if err != nil {
		result = []byte("{}")
	}
	matchingHeaderRequest := len(params.ReqHeader.MockHeader) == matchingHeader

	// Process body matching
	matchingBodyRequest := false
	var foundCase Cases

	for _, vMock := range cases {
		matchingBody := 0
		vMock.Response.FileName = path
		if len(body) == 0 {
			body = core.BindCaseBody(vMock.Body, r)
		}
		for ck, cv := range vMock.Body {
			vs := fmt.Sprintf("%v", cv)
			ks := fmt.Sprintf("%v", body[ck])

			// Check require field value is not empty
			if vs == "*" {
				if body[ck] != nil {
					matchingBody = matchingBody + 1
				}
			}

			// Value matching
			if vs == ks {
				matchingBody = matchingBody + 1
			}
		}

		// Contains value
		matchingBodyRequest = len(vMock.Body) == matchingBody
		if matchingBodyRequest {
			foundCase = vMock
			break
		}
	}

	return CaseMatching{
		IsMatch: matchingBodyRequest && matchingHeaderRequest,
		Result:  result,
		Case:    foundCase,
	}
}

func (u *useCase) ParameterMatching(params Parameters) Matching {
	require := map[string]interface{}{}
	errors := map[string]interface{}{}
	matchingHeader := 0
	matchingBody := 0
	for k, v := range params.ReqBody.MockBody {
		vs := fmt.Sprintf("%v", v)
		ks := fmt.Sprintf("%v", params.ReqBody.HttpBody[k])
		if vs == ks {
			matchingBody = matchingBody + 1
			continue
		}
		if params.ReqBody.HttpBody[k] == nil {
			errors[k] = "Require field " + k
		} else {
			errors[k] = "The " + k + " not match"
		}
	}

	for k, v := range params.ReqHeader.MockHeader {
		vs := fmt.Sprintf("%v", v)
		ks := fmt.Sprintf("%v", params.ReqHeader.HttpHeader[k])
		if vs == ks {
			matchingHeader = matchingHeader + 1
			continue
		}
		if params.ReqHeader.HttpHeader[k] == nil {
			errors[k] = "Require header " + k
		} else {
			errors[k] = "The header " + k + " not match"
		}
	}

	if len(errors) > 0 {
		require["errors"] = errors
		require["message"] = "validation error"
		require["status"] = "error"
	}

	result, err := json.Marshal(require)
	if err != nil {
		result = []byte("{}")
	}

	isMatchHeader := len(params.ReqHeader.MockHeader) == matchingHeader
	isMatchBody := len(params.ReqBody.MockBody) == matchingBody

	return Matching{
		Result:  result,
		IsMatch: isMatchBody && isMatchHeader,
	}
}

func (u *useCase) GetMockResponse(resp Response) []byte {
	if resp.BodyFile != "" {
		bodyFile := fmt.Sprintf(config.MockResponsePath, resp.FileName, resp.BodyFile)
		source, err := ioutil.ReadFile(bodyFile)
		if err != nil {
			return []byte("{}")
		}
		return source
	}
	return []byte(resp.Body)
}

func (u *useCase) ReadSourceRouteYml(routeName string) []byte {
	pattern := status.Pattern()
	filename := fmt.Sprintf(config.MockRouteYmlPath, routeName)
	source, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(pattern)
	}
	return source
}

func NewUseCase() UseCase {
	return &useCase{}
}
