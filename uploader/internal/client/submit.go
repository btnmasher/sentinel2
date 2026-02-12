package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"sentinel2-uploader/internal/logging"
)

func (c *SentinelClient) Submit(payload SubmitPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	c.logger.Debug("submitting report",
		logging.Field("channel_id", payload.ChannelID),
		logging.Field("payload", logging.Truncate(string(body))),
	)

	req, err := http.NewRequest("PUT", c.endpoints.SubmitURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	c.logger.Debugf("PUT %s -> %s", c.endpoints.SubmitURL, resp.Status)

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		formatted := logging.FormatHTTPPayload(data)
		c.logger.Warn("submit rejected",
			logging.Field("status", resp.Status),
			logging.Field("channel_id", payload.ChannelID),
			logging.Field("response", formatted),
		)
		return fmt.Errorf("%s: %s", resp.Status, logging.Truncate(formatted))
	}
	c.logger.Debug("report submit accepted", logging.Field("channel_id", payload.ChannelID))
	return nil
}
