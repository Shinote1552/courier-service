package order_status_changed

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/IBM/sarama"
	"service/internal/entities"
	orderservice "service/internal/service/order"
	"service/pkg/logger"
)

type Handler struct {
	orderService             Service
	log                      handlerLogger
	messageProcessingTimeout time.Duration
}

func New(log handlerLogger, orderService Service, timeout time.Duration) *Handler {
	handlerLog := log.With()

	return &Handler{
		orderService:             orderService,
		log:                      handlerLog,
		messageProcessingTimeout: timeout,
	}
}

func (h *Handler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *Handler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *Handler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				// Messages() закрыт — выходим
				h.log.Info("order.status.changed: claim.Messages() closed, exiting ConsumeClaim")
				return nil
			}

			shouldExit := h.messageProcessing(sess, message)
			if shouldExit {
				return nil
			}

		case <-sess.Context().Done():
			// Сессия закрыта (rebalance или остановка consumer group) — выходим
			h.log.Info("order.status.changed: session context done, exiting ConsumeClaim")
			return nil
		}
	}
}

// messageProcessing обрабатывает одно сообщение из Kafka.
// Возвращает true, если нужно прервать ConsumeClaim (при отмене контекста).
// Возвращает false для продолжения обработки следующих сообщений.
func (h *Handler) messageProcessing(sess sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) bool {
	ctx, cancel := context.WithTimeout(sess.Context(), h.messageProcessingTimeout)
	defer cancel()

	var event createdEvent
	err := json.Unmarshal(message.Value, &event)
	if err != nil {
		h.log.With(
			logger.NewField("error", err),
		).Error("order.status.changed handler received bad message")
		sess.MarkMessage(message, "")
		return false
	}

	msgLog := h.log.With(
		logger.NewField("order", event.OrderID),
		logger.NewField("status", event.Status),
		logger.NewField("offset", message.Offset),
	)

	msgLog.Info("order.status.changed processing")

	status := entities.OrderStatusType(event.Status)
	orderModify := entities.OrderModify{
		ID:     &event.OrderID,
		Status: &status,
	}

	order, err := h.orderService.ProcessOrderStatusChange(ctx, orderModify)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
			msgLog.With(
				logger.NewField("error", err),
			).Warn("order.status.changed handler context cancelled, message will be reprocessed")
			return true

		case errors.Is(err, orderservice.ErrUndefinedStatus):
			msgLog.With(
				logger.NewField("error", err),
			).Warn("order.status.changed handler unknown status for order")

		case errors.Is(err, orderservice.ErrStatusMismatch):
			msgLog.With(
				logger.NewField("error", err),
			).Warn("order.status.changed handler status mismatch for order")

		default:
			msgLog.With(
				logger.NewField("error", err),
			).Warn("order.status.changed handler failed to process order")
		}
		sess.MarkMessage(message, "")
		return false
	}

	// новая дочка с актуальными полями
	msgLog = h.log.With(

		logger.NewField("order", order.ID),
		logger.NewField("event_status", event.Status),
		logger.NewField("current_status", order.Status.String()),
		logger.NewField("offset", message.Offset),
	)
	msgLog.Info("order.status.changed: processed")

	sess.MarkMessage(message, "")
	return false
}
