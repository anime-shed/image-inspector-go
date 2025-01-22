package storage

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/url"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type BlobStorage interface {
	GetImage(ctx context.Context, blobURL string) (image.Image, error)
}

type azureStorage struct {
	client *azblob.Client
}

func NewAzureStorage(accountName string, accountKey string) (BlobStorage, error) {
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, err
	}

	client, err := azblob.NewClientWithSharedKeyCredential(
		fmt.Sprintf("https://%s.blob.core.windows.net", accountName),
		credential,
		nil,
	)

	return &azureStorage{client: client}, nil
}

func (s *azureStorage) GetImage(ctx context.Context, blobURL string) (image.Image, error) {
	parsedURL, err := url.Parse(blobURL)
	if err != nil {
		return nil, fmt.Errorf("invalid blob URL: %w", err)
	}

	containerName := parsedURL.Path[1:] // Remove leading slash
	blobName := parsedURL.Query().Get("blob")

	// Download blob to stream
	downloadResponse, err := s.client.DownloadStream(ctx, containerName, blobName, nil)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	retryReader := downloadResponse.Body
	defer retryReader.Close()

	img, _, err := image.Decode(retryReader)
	return img, err
}
