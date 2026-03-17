package port

import "context"

type StoreClient interface {
	GetStoreOwner(ctx context.Context, storeID string) (ownerID string, commission int, err error)
}
