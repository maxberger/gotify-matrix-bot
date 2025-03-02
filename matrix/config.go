package matrix

import (
	"context"

	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type FakeStateStore struct{}

var _ crypto.StateStore = &FakeStateStore{}

func (fss *FakeStateStore) IsEncrypted(ctx context.Context, roomID id.RoomID) (bool, error) {
	return true, nil
}

func (fss *FakeStateStore) GetEncryptionEvent(ctx context.Context, roomID id.RoomID) (*event.EncryptionEventContent, error) {
	return &event.EncryptionEventContent{
		Algorithm:              id.AlgorithmMegolmV1,
		RotationPeriodMillis:   7 * 24 * 60 * 60 * 1000,
		RotationPeriodMessages: 100,
	}, nil
}

func (fss *FakeStateStore) FindSharedRooms(ctx context.Context, userID id.UserID) ([]id.RoomID, error) {
	return []id.RoomID{}, nil
}

// Easy way to get room members (to find out who to share keys to).
// In real apps, you should cache the member list somewhere and update it based on m.room.member events.
func getUserIDs(ctx context.Context, cli MautrixClientType, roomID id.RoomID) []id.UserID {
	members, err := cli.JoinedMembers(ctx, roomID)
	if err != nil {
		panic(err)
	}
	userIDs := make([]id.UserID, len(members.Joined))
	i := 0
	for userID := range members.Joined {
		userIDs[i] = userID
		i++
	}
	return userIDs
}
