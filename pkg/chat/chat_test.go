package chat_test

// import (
// 	"testing"

// 	"github.com/tonto/gossip/pkg/chat"
// )

// func TestJoin(t *testing.T) {
// 	cases := []struct {
// 		name     string
// 		chat     *chat.Chat
// 		memberID chat.UserID
// 		wantErr  bool
// 	}{
// 		{
// 			name: "test join pvt",
// 			chat: &chat.Chat{
// 				Type:    chat.PvtChat,
// 				Members: []chat.UserID{"fooID", "barID"},
// 			},
// 			memberID: "fooID",
// 			wantErr:  false,
// 		},
// 		{
// 			name: "test join pvt wrong",
// 			chat: &chat.Chat{
// 				Type:    chat.PvtChat,
// 				Members: []chat.UserID{"fooID", "barID"},
// 			},
// 			memberID: "bazID",
// 			wantErr:  true,
// 		},
// 		{
// 			name: "test join chan",
// 			chat: &chat.Chat{
// 				Type: chat.ChanChat,
// 			},
// 			memberID: "fooID",
// 			wantErr:  false,
// 		},
// 		{
// 			name: "test join chan ban",
// 			chat: &chat.Chat{
// 				Type:    chat.ChanChat,
// 				BanList: []chat.UserID{"bazID", "fooID"},
// 			},
// 			memberID: "fooID",
// 			wantErr:  true,
// 		},
// 	}

// 	for _, tc := range cases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			err := tc.chat.Join(tc.memberID)
// 			if (err != nil) != tc.wantErr {
// 				t.Errorf("error = %v, wantErr %v", err, tc.wantErr)
// 				return
// 			}
// 			// if !reflect.DeepEqual(tc.chat.Members, tc.want) {
// 			// 	t.Errorf("Members = %v, want %v", tc.chat.Members, tc.want)
// 			// }
// 		})
// 	}
// }
