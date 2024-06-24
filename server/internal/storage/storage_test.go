package storage

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	responses "shortener/pkg/responses"
)

var urlData, _ = json.Marshal(&responses.Shortener{
	From:           "from",
	ShortUrl:       "short",
	LongUrl:        "long",
	ExpirationDate: time.Now(),
})

var userData, _ = json.Marshal(&responses.Authenticator{
	Id:             "id",
	Name:           "name",
	Email:          "mail",
	HashedPassword: "hash",
})

func TestSotrageSuccess(t *testing.T) {
	b := sarama.NewMockBroker(t, 0)

	b.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(b.Addr(), b.BrokerID()).
			SetLeader("urls", 0, b.BrokerID()).
			SetLeader("users", 0, b.BrokerID()).
			SetController(b.BrokerID()),
		"FindCoordinatorRequest": sarama.NewMockFindCoordinatorResponse(t).
			SetCoordinator(sarama.CoordinatorGroup, "mock", b),
		"ApiVersionsRequest": sarama.NewMockApiVersionsResponse(t),
		"OffsetRequest": sarama.NewMockOffsetResponse(t).
			SetOffset("urls", 0, sarama.OffsetOldest, 0).
			SetOffset("urls", 0, sarama.OffsetNewest, 1).
			SetOffset("users", 0, sarama.OffsetOldest, 0).
			SetOffset("users", 0, sarama.OffsetNewest, 1),
		"HeartbeatRequest": sarama.NewMockHeartbeatResponse(t),
		"JoinGroupRequest": sarama.NewMockSequence(
			sarama.NewMockJoinGroupResponse(t).
				SetError(sarama.ErrOffsetsLoadInProgress),
			sarama.NewMockJoinGroupResponse(
				t,
			).SetGroupProtocol(sarama.RangeBalanceStrategyName),
		),
		"SyncGroupRequest": sarama.NewMockSequence(
			sarama.NewMockSyncGroupResponse(t).
				SetError(sarama.ErrOffsetsLoadInProgress),
			sarama.NewMockSyncGroupResponse(t).SetMemberAssignment(
				&sarama.ConsumerGroupMemberAssignment{
					Version: 0,
					Topics: map[string][]int32{
						"urls":  {0},
						"users": {0},
					},
				}),
		),
		"OffsetFetchRequest": sarama.NewMockOffsetFetchResponse(t).
			SetOffset("mock", "users", 0, 0, "", sarama.ErrNoError).
			SetError(sarama.ErrNoError).
			SetOffset("mock", "urls", 0, 0, "", sarama.ErrNoError).
			SetError(sarama.ErrNoError),
		"FetchRequest": sarama.NewMockSequence(
			sarama.NewMockFetchResponse(t, 1).
				SetMessage("users", 0, 0, sarama.ByteEncoder(userData)).
				SetMessage("urls", 0, 0, sarama.ByteEncoder(urlData)),
			// sarama.NewMockFetchResponse(t, 1),
		),
	})

	defer b.Close()

	address := b.Addr()

	urlsModel := NewMockUrls(t)
	urlsModel.EXPECT().
		Insert(context.TODO(), mock.MatchedBy(func(rr []*responses.Shortener) bool {
			if len(rr) != 1 {
				return false
			}
			r := rr[0]
			if r.From == "from" && r.LongUrl == "long" &&
				r.ShortUrl == "short" {
				return true
			}
			return false
		})).
		Once().
		Return()

	usersModel := NewMockUsers(t)
	usersModel.EXPECT().
		Insert(context.TODO(), mock.MatchedBy(func(rr []*responses.Authenticator) bool {
			if len(rr) != 1 {
				return false
			}
			r := rr[0]
			if r.Name == "name" && r.Id == "id" && r.Email == "mail" &&
				r.HashedPassword == "hash" {
				return true
			}
			return false
		})).
		Once().
		Return()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h, err := New(
		WithLogger(&log.Logger),
		WithContext(ctx),
		WithUrlsModel(urlsModel),
		WithUsersModel(usersModel),
		WithUrlsTopic("urls"),
		WithUsersTopic("users"),
	)
	assert.Nil(t, err)

	conf := sarama.NewConfig()
	conf.Consumer.Offsets.AutoCommit.Enable = false
	conf.Consumer.Offsets.Initial = sarama.OffsetOldest

	err = conf.Validate()
	assert.Nil(t, err)

	log.Info().Msg(address)
	g, err := sarama.NewConsumerGroup([]string{address}, "mock", conf)
	assert.Nil(t, err)
	defer g.Close()

	go func() {
		<-ctx.Done()
		g.Close()
	}()

	err = g.Consume(context.Background(), []string{"urls", "users"}, h)
	log.Printf("%v", err)
	assert.Nil(t, err)
}
