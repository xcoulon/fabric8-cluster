package controller

import (
	"github.com/fabric8-services/fabric8-cluster/app"
	"github.com/fabric8-services/fabric8-cluster/application"
	cluster "github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/fabric8-services/fabric8-common/auth"
	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/fabric8-common/log"

	"fmt"

	"github.com/fabric8-services/fabric8-cluster/cluster/service"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// ClustersController implements the clusters resource.
type ClustersController struct {
	*goa.Controller
	app application.Application
}

// NewClustersController creates a clusters controller.
func NewClustersController(service *goa.Service, app application.Application) *ClustersController {
	return &ClustersController{
		Controller: service.NewController("ClustersController"),
		app:        app,
	}
}

// List returns the list of available clusters.
func (c *ClustersController) List(ctx *app.ListClustersContext) error {
	clusters, err := c.app.ClusterService().List(ctx)
	if err != nil {
		return app.JSONErrorResponse(ctx, err)
	}
	var data []*app.ClusterData
	for _, c := range clusters {
		clusterData := &app.ClusterData{
			Name:              c.Name,
			APIURL:            httpsupport.AddTrailingSlashToURL(c.URL),
			ConsoleURL:        httpsupport.AddTrailingSlashToURL(c.ConsoleURL),
			MetricsURL:        httpsupport.AddTrailingSlashToURL(c.MetricsURL),
			LoggingURL:        httpsupport.AddTrailingSlashToURL(c.LoggingURL),
			AppDNS:            c.AppDNS,
			Type:              c.Type,
			CapacityExhausted: c.CapacityExhausted,
		}
		data = append(data, clusterData)
	}
	return ctx.OK(&app.ClusterList{
		Data: data,
	})
}

// ListForAuthClient returns the list of available clusters with full configuration including Auth client data.
// To be used by Auth service only
func (c *ClustersController) ListForAuthClient(ctx *app.ListForAuthClientClustersContext) error {
	clusters, err := c.app.ClusterService().ListForAuth(ctx)
	if err != nil {
		return app.JSONErrorResponse(ctx, err)
	}
	var data []*app.FullClusterData
	for _, c := range clusters {
		encrypted := c.SATokenEncrypted
		clusterData := &app.FullClusterData{
			Name:                   c.Name,
			APIURL:                 httpsupport.AddTrailingSlashToURL(c.URL),
			ConsoleURL:             httpsupport.AddTrailingSlashToURL(c.ConsoleURL),
			MetricsURL:             httpsupport.AddTrailingSlashToURL(c.MetricsURL),
			LoggingURL:             httpsupport.AddTrailingSlashToURL(c.LoggingURL),
			AppDNS:                 c.AppDNS,
			Type:                   c.Type,
			CapacityExhausted:      c.CapacityExhausted,
			AuthClientDefaultScope: c.AuthDefaultScope,
			AuthClientID:           c.AuthClientID,
			AuthClientSecret:       c.AuthClientSecret,
			SaTokenEncrypted:       &encrypted,
			ServiceAccountToken:    c.SAToken,
			ServiceAccountUsername: c.SAUsername,
			TokenProviderID:        c.TokenProviderID,
		}
		data = append(data, clusterData)
	}
	return ctx.OK(&app.FullClusterList{
		Data: data,
	})
}

// Show returns the single clusters.
func (c *ClustersController) Show(ctx *app.ShowClustersContext) error {
	// authorization is checked at the service level for more consistency accross the codebase.
	clustr, err := c.app.ClusterService().Load(ctx, ctx.ClusterID)
	if err != nil {
		return app.JSONErrorResponse(ctx, err)
	}
	return ctx.OK(&app.ClusterSingle{
		Data: &app.ClusterData{
			Name:              clustr.Name,
			APIURL:            httpsupport.AddTrailingSlashToURL(clustr.URL),
			ConsoleURL:        httpsupport.AddTrailingSlashToURL(clustr.ConsoleURL),
			MetricsURL:        httpsupport.AddTrailingSlashToURL(clustr.MetricsURL),
			LoggingURL:        httpsupport.AddTrailingSlashToURL(clustr.LoggingURL),
			AppDNS:            clustr.AppDNS,
			Type:              clustr.Type,
			CapacityExhausted: clustr.CapacityExhausted,
		},
	})
}

// ShowForAuthClient returns the cluster with full configuration including Auth client data.
// To be used by Auth service only
func (c *ClustersController) ShowForAuthClient(ctx *app.ShowForAuthClientClustersContext) error {
	// authorization is checked at the service level for more consistency accross the codebase.
	clustr, err := c.app.ClusterService().LoadForAuth(ctx, ctx.ClusterID)
	if err != nil {
		return app.JSONErrorResponse(ctx, err)
	}
	encrypted := clustr.SATokenEncrypted
	return ctx.OK(&app.FullClusterSingle{
		Data: &app.FullClusterData{
			Name:                   clustr.Name,
			APIURL:                 httpsupport.AddTrailingSlashToURL(clustr.URL),
			ConsoleURL:             httpsupport.AddTrailingSlashToURL(clustr.ConsoleURL),
			MetricsURL:             httpsupport.AddTrailingSlashToURL(clustr.MetricsURL),
			LoggingURL:             httpsupport.AddTrailingSlashToURL(clustr.LoggingURL),
			AppDNS:                 clustr.AppDNS,
			Type:                   clustr.Type,
			CapacityExhausted:      clustr.CapacityExhausted,
			AuthClientDefaultScope: clustr.AuthDefaultScope,
			AuthClientID:           clustr.AuthClientID,
			AuthClientSecret:       clustr.AuthClientSecret,
			SaTokenEncrypted:       &encrypted,
			ServiceAccountToken:    clustr.SAToken,
			ServiceAccountUsername: clustr.SAUsername,
			TokenProviderID:        clustr.TokenProviderID,
		},
	})
}

// Create creates a new cluster configuration for later use
func (c *ClustersController) Create(ctx *app.CreateClustersContext) error {
	// check that the token belongs to a user
	if !auth.IsSpecificServiceAccount(ctx, auth.ToolChainOperator) {
		log.Error(ctx, nil, "unauthorized access to cluster info")
		return app.JSONErrorResponse(ctx, errors.NewUnauthorizedError("unauthorized access to cluster info"))
	}
	clustr := cluster.Cluster{
		Name:             ctx.Payload.Data.Name,
		Type:             ctx.Payload.Data.Type,
		URL:              ctx.Payload.Data.APIURL,
		AppDNS:           ctx.Payload.Data.AppDNS,
		SAToken:          ctx.Payload.Data.ServiceAccountToken,
		SAUsername:       ctx.Payload.Data.ServiceAccountUsername,
		AuthClientID:     ctx.Payload.Data.AuthClientID,
		AuthClientSecret: ctx.Payload.Data.AuthClientSecret,
		AuthDefaultScope: ctx.Payload.Data.AuthClientDefaultScope,
	}
	if ctx.Payload.Data.ConsoleURL != nil {
		clustr.ConsoleURL = *ctx.Payload.Data.ConsoleURL
	}
	if ctx.Payload.Data.LoggingURL != nil {
		clustr.LoggingURL = *ctx.Payload.Data.LoggingURL
	}
	if ctx.Payload.Data.MetricsURL != nil {
		clustr.MetricsURL = *ctx.Payload.Data.MetricsURL
	}
	if ctx.Payload.Data.CapacityExhausted != nil {
		clustr.CapacityExhausted = *ctx.Payload.Data.CapacityExhausted
	}
	if ctx.Payload.Data.TokenProviderID != nil {
		clustr.TokenProviderID = *ctx.Payload.Data.TokenProviderID
	}
	clusterSvc := c.app.ClusterService()
	err := clusterSvc.CreateOrSaveCluster(ctx, &clustr)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"error": err,
		}, "error while creating new cluster configuration")
		return app.JSONErrorResponse(ctx, err)
	}
	ctx.ResponseData.Header().Set("Location", app.ClustersHref(clustr.ClusterID.String()))
	return ctx.Created()
}

// Delete deletes the cluster identified by the `clusterID` param
func (c *ClustersController) Delete(ctx *app.DeleteClustersContext) error {
	err := c.app.ClusterService().Delete(ctx, ctx.ClusterID)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"error": err,
		}, "error while deleting a cluster configuration")
		return app.JSONErrorResponse(ctx, err)
	}
	return ctx.NoContent()
}

// LinkIdentityToCluster populates Identity Cluster relationship
func (c *ClustersController) LinkIdentityToCluster(ctx *app.LinkIdentityToClusterClustersContext) error {
	if !auth.IsSpecificServiceAccount(ctx, auth.Auth) {
		log.Error(ctx, nil, "the account is not authorized to create identity cluster relationship")
		return app.JSONErrorResponse(ctx, errors.NewUnauthorizedError("account not authorized to create identity cluster relationship"))
	}

	identityID, err := uuid.FromString(ctx.Payload.IdentityID)
	if err != nil {
		return app.JSONErrorResponse(ctx, errors.NewBadParameterErrorFromString(fmt.Sprintf("identity-id %s is not a valid UUID", identityID)))
	}

	clusterURL := ctx.Payload.ClusterURL
	if err := service.ValidateURL(&clusterURL); err != nil {
		return app.JSONErrorResponse(ctx, errors.NewBadParameterErrorFromString(fmt.Sprintf("cluster-url '%s' is invalid", clusterURL)))
	}

	// ignoreIfAlreadyExisted by default true
	ignore := true
	if ignoreIfExists := ctx.Payload.IgnoreIfAlreadyExists; ignoreIfExists != nil {
		ignore = *ignoreIfExists
	}
	if err := c.app.ClusterService().LinkIdentityToCluster(ctx, identityID, clusterURL, ignore); err != nil {
		log.Error(ctx, map[string]interface{}{
			"error": err,
		}, "error while linking identity-id %s to cluster with url %s", identityID, clusterURL)
		return app.JSONErrorResponse(ctx, err)
	}

	return ctx.NoContent()
}

// RemoveIdentityToClusterLink removes Identity Cluster relationship
func (c *ClustersController) RemoveIdentityToClusterLink(ctx *app.RemoveIdentityToClusterLinkClustersContext) error {
	if !auth.IsSpecificServiceAccount(ctx, auth.Auth) {
		log.Error(ctx, nil, "the account is not authorized to remove identity cluster relationship")
		return app.JSONErrorResponse(ctx, errors.NewUnauthorizedError("account not authorized to remove identity cluster relationship"))
	}

	identityID, err := uuid.FromString(ctx.Payload.IdentityID)
	if err != nil {
		return app.JSONErrorResponse(ctx, errors.NewBadParameterErrorFromString(fmt.Sprintf("identity-id %s is not a valid UUID", identityID)))
	}

	clusterURL := ctx.Payload.ClusterURL
	if err := service.ValidateURL(&clusterURL); err != nil {
		return app.JSONErrorResponse(ctx, errors.NewBadParameterErrorFromString(fmt.Sprintf("cluster-url '%s' is invalid", clusterURL)))
	}

	if err := c.app.ClusterService().RemoveIdentityToClusterLink(ctx, identityID, clusterURL); err != nil {
		log.Error(ctx, map[string]interface{}{
			"error": err,
		}, "error while removing link of identity-id %s to cluster with url %s", identityID, clusterURL)
		return app.JSONErrorResponse(ctx, err)
	}

	return ctx.NoContent()
}
