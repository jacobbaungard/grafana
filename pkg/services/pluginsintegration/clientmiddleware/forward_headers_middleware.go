package clientmiddleware

import (
	"context"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"

	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/plugins"
	"github.com/grafana/grafana/pkg/services/contexthandler"
	"github.com/grafana/grafana/pkg/services/datasources"
)

// NewForwardHeadersMiddleware creates a new plugins.ClientMiddleware
// will forward any ensure configured headers are forwarded
func NewForwardHeadersMiddleware() plugins.ClientMiddleware {
	return plugins.ClientMiddlewareFunc(func(next plugins.Client) plugins.Client {
		return &ForwardHeadersMiddleware{
			baseMiddleware: baseMiddleware{
				next: next,
			},
			log: log.New("Forward Headers Middleware"),
		}
	})
}

type ForwardHeadersMiddleware struct {
	baseMiddleware
	log log.Logger
}

func (m *ForwardHeadersMiddleware) forwardHeaders(ctx context.Context, pCtx backend.PluginContext, h backend.ForwardHTTPHeaders) {
	reqCtx := contexthandler.FromContext(ctx)
	// if no HTTP request context skip middleware
	if h == nil || reqCtx == nil || reqCtx.Req == nil || reqCtx.SignedInUser == nil {
		return
	}

	if pCtx.DataSourceInstanceSettings != nil {
		settings := pCtx.DataSourceInstanceSettings
		jsonDataBytes, err := simplejson.NewJson(settings.JSONData)
		if err != nil {
			return
		}

		ds := &datasources.DataSource{
			ID:       settings.ID,
			OrgID:    pCtx.OrgID,
			JsonData: jsonDataBytes,
			Updated:  settings.Updated,
		}

		headersToFoward := ds.ForwardHeaders()
		if len(headersToFoward) == 0 {
			m.log.Warn("No headers to forward")
			return
		}

		var req *http.Request
		if reqCtx != nil {
			req = reqCtx.Req
		}
		for _, k := range headersToFoward {
			if _, ok := req.Header[k]; ok {
				m.log.Debug("Fowarding header: ", "header", k)
				h.SetHTTPHeader(k, req.Header.Get(k))
			}
		}
	}

}

func (m *ForwardHeadersMiddleware) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	if req == nil {
		return m.next.QueryData(ctx, req)
	}

	m.forwardHeaders(ctx, req.PluginContext, req)

	return m.next.QueryData(ctx, req)
}

func (m *ForwardHeadersMiddleware) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	if req == nil {
		return m.next.CallResource(ctx, req, sender)
	}

	m.forwardHeaders(ctx, req.PluginContext, req)

	return m.next.CallResource(ctx, req, sender)
}

func (m *ForwardHeadersMiddleware) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	if req == nil {
		return m.next.CheckHealth(ctx, req)
	}

	m.forwardHeaders(ctx, req.PluginContext, req)

	return m.next.CheckHealth(ctx, req)
}
