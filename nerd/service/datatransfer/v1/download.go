package v1datatransfer

import (
	"context"
	"time"

	v1batch "github.com/nerdalize/nerd/nerd/client/batch/v1"
	v1payload "github.com/nerdalize/nerd/nerd/client/batch/v1/payload"
	v1data "github.com/nerdalize/nerd/nerd/service/datatransfer/v1/client"
	"github.com/pkg/errors"
)

//DownloadConfig is the config for Download operations
type DownloadConfig struct {
	BatchClient v1batch.ClientInterface
	DataOps     v1data.DataOps
	LocalDir    string
	ProjectID   string
	DatasetID   string
	Concurrency int
	ProgressCh  chan<- int64
}

//Download downloads a dataset or fails if it is still being uploaded
func Download(ctx context.Context, conf DownloadConfig) error {
	if conf.ProgressCh != nil {
		defer close(conf.ProgressCh)
	}
	var ds v1payload.DatasetSummary
	for {
		out, err := conf.BatchClient.DescribeDataset(conf.ProjectID, conf.DatasetID)
		if err != nil {
			return errors.Wrap(err, "failed to get dataset")
		}
		ds = out.DatasetSummary
		if ds.UploadStatus == v1payload.DatasetUploadStatusSuccess {
			break
		}
		if ds.UploadStatus == v1payload.DatasetUploadStatusUploading && ds.UploadExpire < time.Now().Unix() {
			return errors.Errorf("cannot start download, because the upload timed out")
		}
		wait := ds.UploadExpire - time.Now().Unix()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second * time.Duration(wait)):
		}
	}
	dataClient := v1data.NewClient(conf.DataOps)
	down := &downloadProcess{
		dataClient:  dataClient,
		dataset:     ds,
		localDir:    conf.LocalDir,
		concurrency: conf.Concurrency,
		progressCh:  conf.ProgressCh,
	}
	return down.start(ctx)
}

//GetRemoteDatasetSize gets the size of a dataset from the metadata object
func GetRemoteDatasetSize(ctx context.Context, batchClient *v1batch.Client, dataOps v1data.DataOps, projectID, datasetID string) (int64, error) {
	dataClient := v1data.NewClient(dataOps)
	ds, err := batchClient.DescribeDataset(projectID, datasetID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get dataset")
	}
	metadata, err := dataClient.MetadataDownload(ctx, ds.Bucket, ds.DatasetRoot)
	if err != nil {
		return 0, errors.Wrap(err, "failed to download metadata")
	}
	return metadata.Size, nil
}
