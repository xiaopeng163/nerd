package provider

import (
	"crypto/ecdsa"

	"github.com/nerdalize/nerd/nerd/client/credentials"
	"github.com/pkg/errors"
)

//ChainProvider provides nerdalize credentials based on multiple providers. The given providers are tried in sequential order.
type ChainProvider struct {
	Providers []credentials.Provider
	curr      credentials.Provider
}

func NewChainCredentials(pub *ecdsa.PublicKey, providers ...credentials.Provider) *credentials.NerdAPI {
	return credentials.NewNerdAPI(pub, &ChainProvider{
		Providers: providers,
	})
}

// Retrieve returns the credentials value or error if no provider returned
// without error.
//
// If a provider is found it will be cached and any calls to IsExpired()
// will return the expired state of the cached provider.
func (c *ChainProvider) Retrieve(pub *ecdsa.PublicKey) (*credentials.NerdAPIValue, error) {
	var errs []error
	for _, p := range c.Providers {
		creds, err := p.Retrieve(pub)
		if err == nil {
			c.curr = p
			return creds, nil
		}
		errs = append(errs, err)
	}
	c.curr = nil

	return nil, errors.Errorf("could not retreive token from any provider: %v", errs)
}

// IsExpired will returned the expired state of the currently cached provider
// if there is one.  If there is no current provider, true will be returned.
func (c *ChainProvider) IsExpired() bool {
	if c.curr != nil {
		return c.curr.IsExpired()
	}

	return true
}