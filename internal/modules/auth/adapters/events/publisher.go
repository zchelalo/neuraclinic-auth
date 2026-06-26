package events

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	authv1 "github.com/zchelalo/neuraclinic-auth/gen/go/auth/v1"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type NoopPublisher struct{}

func NewNoopPublisher() *NoopPublisher {
	return &NoopPublisher{}
}

func (p *NoopPublisher) PublishPasswordResetRequested(_ context.Context, _ ports.PasswordResetRequestedEvent) error {
	return nil
}

func (p *NoopPublisher) Close() error {
	return nil
}

type RabbitPublisher struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	exchange   string
	routingKey string
}

func NewRabbitPublisher(url, exchange, routingKey string) (*RabbitPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("declare rabbitmq exchange: %w", err)
	}

	return &RabbitPublisher{
		conn:       conn,
		channel:    ch,
		exchange:   exchange,
		routingKey: routingKey,
	}, nil
}

func (p *RabbitPublisher) PublishPasswordResetRequested(ctx context.Context, event ports.PasswordResetRequestedEvent) error {
	body, err := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: true,
	}.Marshal(&authv1.PasswordResetRequestedEvent{
		EventId:   event.EventID,
		UserId:    event.UserID,
		Email:     event.Email,
		Otp:       event.OTP,
		Language:  event.Language,
		ExpiresAt: timestamppb.New(event.ExpiresAt),
		RequestId: event.RequestID,
		TraceId:   event.TraceID,
	})
	if err != nil {
		return err
	}

	return p.channel.PublishWithContext(ctx, p.exchange, p.routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (p *RabbitPublisher) Close() error {
	var err error
	if p.channel != nil {
		err = p.channel.Close()
	}
	if p.conn != nil {
		if closeErr := p.conn.Close(); err == nil {
			err = closeErr
		}
	}
	return err
}
