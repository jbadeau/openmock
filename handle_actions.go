package openmock

import (
	"time"

	"github.com/fatih/structs"
	"github.com/labstack/echo"
	"github.com/parnurzeal/gorequest"
	"github.com/sirupsen/logrus"
)

// DoActions do actions based on the context
func (ms MocksArray) DoActions(c *Context) error {
	for _, m := range ms {
		if !c.MatchCondition(m.Expect.Condition) {
			continue
		}
		if err := m.DoActions(c); err != nil {
			return nil
		}
	}
	return nil
}

// DoActions runs all the actions
func (m *Mock) DoActions(c *Context) error {
	for _, a := range m.Actions {
		if err := m.doAction(c, a); err != nil {
			return err
		}
	}
	return nil
}

func (m *Mock) doAction(c *Context, a Action) (err error) {
	var action string

	defer func() {
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"err":    err,
				"action": action,
			}).Errorf("failed to do action")
		}
	}()

	if err := m.doActionSleep(c, a); err != nil {
		action = "sleep"
		return err
	}
	if err := m.doActionRedis(c, a); err != nil {
		action = "redis"
		return err
	}
	if err := m.doActionPublishKafka(c, a); err != nil {
		action = "publish_kafka"
		return err
	}
	if err := m.doActionPublishAMQP(c, a); err != nil {
		action = "publish_amqp"
		return err
	}
	if err := m.doActionSendHTTP(c, a); err != nil {
		action = "send_http"
		return err
	}

	if err := m.doActionReplyHTTP(c, a); err != nil {
		action = "reply_http"
		return err
	}

	return nil
}

func (m *Mock) doActionSendHTTP(c *Context, a Action) error {
	if structs.IsZero(a.ActionSendHTTP) {
		return nil
	}

	sendHTTP := a.ActionSendHTTP

	bodyStr, err := c.Render(sendHTTP.Body)
	if err != nil {
		return err
	}

	urlStr, err := c.Render(sendHTTP.URL)
	if err != nil {
		return err
	}

	request := gorequest.New().
		SetDebug(true).
		CustomMethod(sendHTTP.Method, urlStr)

	for k, v := range sendHTTP.Headers {
		request.Set(k, v)
	}

	_, _, errs := request.Send(bodyStr).End()
	if len(errs) != 0 {
		return errs[0]
	}
	return nil
}

func (m *Mock) doActionReplyHTTP(c *Context, a Action) error {
	if structs.IsZero(a.ActionReplyHTTP) {
		return nil
	}
	h := a.ActionReplyHTTP
	ec := c.HTTPContext
	contentType := echo.MIMEApplicationJSON // default to JSON
	if ct, ok := h.Headers[echo.HeaderContentType]; ok {
		contentType = ct
	}
	for k, v := range h.Headers {
		ec.Response().Header().Set(k, v)
	}
	msg, err := c.Render(h.Body)
	if err != nil {
		logrus.WithField("err", err).Error("failed to render template for http")
		return err
	}
	return ec.Blob(h.StatusCode, contentType, []byte(msg))
}

func (m *Mock) doActionRedis(c *Context, a Action) error {
	if len(a.ActionRedis) == 0 {
		return nil
	}
	for _, cmd := range a.ActionRedis {
		_, err := c.Render(cmd)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Mock) doActionSleep(c *Context, a Action) error {
	if structs.IsZero(a.ActionSleep) {
		return nil
	}
	time.Sleep(a.ActionSleep.Duration)
	return nil
}

func (m *Mock) doActionPublishKafka(c *Context, a Action) error {
	if structs.IsZero(a.ActionPublishKafka) {
		return nil
	}

	k := a.ActionPublishKafka
	msg := k.Payload
	msg, err := c.Render(msg)
	if err != nil {
		logrus.WithField("err", err).Error("failed to render template for kafka payload")
		return err
	}
	err = c.om.kafkaClient.sendMessage(k.Topic, []byte(msg))
	if err != nil {
		logrus.WithField("err", err).Error("failed to publish to kafka")
	}
	return err
}

func (m *Mock) doActionPublishAMQP(c *Context, a Action) error {
	if structs.IsZero(a.ActionPublishAMQP) {
		return nil
	}
	w := a.ActionPublishAMQP
	msg, err := c.Render(w.Payload)
	if err != nil {
		logrus.WithField("err", err).Error("failed to render template for amqp")
		return err
	}
	publishToAMQP(
		c.om.AMQPURL,
		w.Exchange,
		w.RoutingKey,
		msg,
	)
	return nil
}
