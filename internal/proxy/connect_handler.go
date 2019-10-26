package proxy

import (
	"context"
	"encoding/base64"

	"github.com/centrifugal/centrifuge"
)

// ConnectHandlerConfig ...
type ConnectHandlerConfig struct {
	Proxy ConnectProxy
}

// ConnectHandler ...
type ConnectHandler struct {
	config ConnectHandlerConfig
}

// NewConnectHandler ...
func NewConnectHandler(c ConnectHandlerConfig) *ConnectHandler {
	return &ConnectHandler{
		config: c,
	}
}

// Handle returns connecting handler func.
func (h *ConnectHandler) Handle(node *centrifuge.Node) func(ctx context.Context, t centrifuge.TransportInfo, e centrifuge.ConnectEvent) centrifuge.ConnectReply {
	return func(ctx context.Context, t centrifuge.TransportInfo, e centrifuge.ConnectEvent) centrifuge.ConnectReply {
		if e.Token != "" {
			// As soon as token provided we do not try to proxy connect to application backend.
			return centrifuge.ConnectReply{
				Credentials: nil,
			}
		}

		connectRep, err := h.config.Proxy.ProxyConnect(ctx, ConnectRequest{
			ClientID:  e.ClientID,
			Transport: t,
			Data:      e.Data,
		})
		if err != nil {
			node.Log(centrifuge.NewLogEntry(centrifuge.LogLevelError, "error proxying connect", map[string]interface{}{"client": e.ClientID, "error": err.Error()}))
			return centrifuge.ConnectReply{
				Error: centrifuge.ErrorInternal,
			}
		}
		if connectRep.Disconnect != nil {
			return centrifuge.ConnectReply{
				Disconnect: connectRep.Disconnect,
			}
		}
		if connectRep.Error != nil {
			return centrifuge.ConnectReply{
				Error: connectRep.Error,
			}
		}

		credentials := connectRep.Result
		if credentials == nil {
			return centrifuge.ConnectReply{
				Credentials: nil,
			}
		}

		var info []byte
		if t.Encoding() == "json" {
			info = credentials.Info
		} else {
			if credentials.Base64Info != "" {
				decodedInfo, err := base64.StdEncoding.DecodeString(credentials.Base64Info)
				if err != nil {
					node.Log(centrifuge.NewLogEntry(centrifuge.LogLevelError, "error decoding base64 info", map[string]interface{}{"client": e.ClientID, "error": err.Error()}))
					return centrifuge.ConnectReply{
						Error: centrifuge.ErrorInternal,
					}
				}
				info = decodedInfo
			}
		}

		return centrifuge.ConnectReply{
			Credentials: &centrifuge.Credentials{
				UserID:   credentials.UserID,
				ExpireAt: credentials.ExpireAt,
				Info:     info,
			},
		}
	}
}