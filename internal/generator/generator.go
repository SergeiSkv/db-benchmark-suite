package generator

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

type Event struct {
	ID        string
	UserID    int64
	EventType string
	Payload   string
	CreatedAt time.Time
}

type Generator struct {
	totalEvents int
	batchSize   int
	current     int
	rand        *rand.Rand
}

var eventTypes = []string{
	"page_view",
	"button_click",
	"form_submit",
	"api_call",
	"error",
	"login",
	"logout",
	"purchase",
	"add_to_cart",
	"search",
}

func New(totalEvents, batchSize int) *Generator {
	return &Generator{
		totalEvents: totalEvents,
		batchSize:   batchSize,
		current:     0,
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (g *Generator) Generate() <-chan []Event {
	ch := make(chan []Event, 10)

	go func() {
		defer close(ch)

		for g.current < g.totalEvents {
			remaining := g.totalEvents - g.current

			size := g.batchSize
			if remaining < size {
				size = remaining
			}

			batch := make([]Event, size)
			for i := 0; i < size; i++ {
				batch[i] = g.generateEvent()
			}

			ch <- batch

			g.current += size
		}
	}()

	return ch
}

func (g *Generator) generateEvent() Event {
	// Generate realistic timestamps (last 90 days) with exponential bias toward recent data
	const lambda = 0.05 // rate parameter — lower = more spread, higher = more recent

	daysAgo := int(-math.Log(1-g.rand.Float64()) / lambda)
	if daysAgo > 89 {
		daysAgo = 89
	}

	hoursAgo := g.rand.Intn(24)
	minutesAgo := g.rand.Intn(60)
	secondsAgo := g.rand.Intn(60)

	createdAt := time.Now().
		AddDate(0, 0, -daysAgo).
		Add(-time.Duration(hoursAgo) * time.Hour).
		Add(-time.Duration(minutesAgo) * time.Minute).
		Add(-time.Duration(secondsAgo) * time.Second)

	return Event{
		ID:        fmt.Sprintf("evt_%d_%d", createdAt.UnixNano(), g.rand.Int63()),
		UserID:    g.rand.Int63n(1000000), // 1M unique users
		EventType: eventTypes[g.rand.Intn(len(eventTypes))],
		Payload:   g.generatePayload(),
		CreatedAt: createdAt,
	}
}

func (g *Generator) generatePayload() string {
	// Generate realistic JSON payload
	templates := []string{
		`{"page": "/home", "referrer": "google.com", "session_id": "%s"}`,
		`{"button": "checkout", "product_id": %d, "price": %.2f}`,
		`{"form": "contact", "fields": %d, "success": %t}`,
		`{"endpoint": "/api/users", "method": "POST", "status": %d}`,
		`{"error_code": "ERR_%d", "message": "Connection timeout", "retry": %d}`,
	}

	idx := g.rand.Intn(len(templates))
	template := templates[idx]

	switch idx {
	case 0:
		return fmt.Sprintf(template, g.randomString(32))
	case 1:
		return fmt.Sprintf(template, g.rand.Int63n(10000), g.rand.Float64()*1000)
	case 2:
		return fmt.Sprintf(template, g.rand.Intn(20), g.rand.Intn(2) == 1)
	case 3:
		return fmt.Sprintf(template, 200+g.rand.Intn(299))
	default:
		return fmt.Sprintf(template, g.rand.Intn(9999), g.rand.Intn(5))
	}
}

func (g *Generator) randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, length)

	for i := 0; i < length; i++ {
		b[i] = charset[g.rand.Intn(len(charset))]
	}

	return string(b)
}
