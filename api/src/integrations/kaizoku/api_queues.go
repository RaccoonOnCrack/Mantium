package kaizoku

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/diogovalentte/mantium/api/src/util"
)

func (k *Kaizoku) GetQueues() ([]*Queue, error) {
	errorContext := "Error while getting queues"

	url := fmt.Sprintf("%s/bull/queues/api/queues", k.Address)
	resp, err := k.Request(http.MethodGet, url, nil)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}
	defer resp.Body.Close()
	err = validateResponse(resp)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}

	var queues getQueuesResponse
	err = json.NewDecoder(resp.Body).Decode(&queues)
	if err != nil {
		return nil, util.AddErrorContext(err, util.AddErrorContext(fmt.Errorf("Error while decoding response body"), errorContext).Error())
	}

	return queues.Queues, nil
}

func (k *Kaizoku) GetQueue(queueName string) (*Queue, error) {
	errorContext := "Error while getting queue '%s'"

	queues, err := k.GetQueues()
	if err != nil {
		return nil, util.AddErrorContext(err, fmt.Sprintf(errorContext, queueName))
	}

	var queue *Queue
	for _, q := range queues {
		if q.Name == queueName {
			queue = q
			break
		}
	}

	if queue == nil {
		return nil, util.AddErrorContext(fmt.Errorf("Queue not found"), fmt.Sprintf(errorContext, queueName))
	}

	return queue, nil
}

func (k *Kaizoku) RetryFailedFixOutOfSyncChaptersQueueJobs() error {
	errorContext := "Error while retrying failed fix out of sync chapters queue jobs"

	url := fmt.Sprintf("%s/bull/queues/api/queues/fixOutOfSyncChaptersQueue/retry/failed", k.Address)
	resp, err := k.Request(http.MethodPut, url, nil)
	if err != nil {
		return util.AddErrorContext(err, errorContext)
	}
	defer resp.Body.Close()
	err = validateResponse(resp)
	if err != nil {
		return util.AddErrorContext(err, errorContext)
	}

	return nil
}

type getQueuesResponse struct {
	Queues []*Queue `json:"queues"`
}
