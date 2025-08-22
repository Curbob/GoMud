package auctions

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// AuctionCopyoverState represents the auction system state during copyover
type AuctionCopyoverState struct {
	HasActiveAuction bool               `json:"has_active_auction"`
	ActiveAuction    *AuctionItemState  `json:"active_auction,omitempty"`
	PastAuctions     []PastAuctionState `json:"past_auctions,omitempty"`
}

// AuctionItemState represents an active auction's state
type AuctionItemState struct {
	ItemId            int       `json:"item_id"`
	ItemName          string    `json:"item_name"`
	ItemDescription   string    `json:"item_description"`
	SellerUserId      int       `json:"seller_user_id"`
	SellerName        string    `json:"seller_name"`
	Anonymous         bool      `json:"anonymous"`
	EndTime           time.Time `json:"end_time"`
	MinimumBid        int       `json:"minimum_bid"`
	HighestBid        int       `json:"highest_bid"`
	HighestBidUserId  int       `json:"highest_bid_user_id,omitempty"`
	HighestBidderName string    `json:"highest_bidder_name,omitempty"`
	LastUpdate        time.Time `json:"last_update"`
}

// PastAuctionState represents a completed auction for history
type PastAuctionState struct {
	ItemName   string    `json:"item_name"`
	WinningBid int       `json:"winning_bid"`
	Anonymous  bool      `json:"anonymous"`
	SellerName string    `json:"seller_name"`
	BuyerName  string    `json:"buyer_name"`
	EndTime    time.Time `json:"end_time"`
}

var auctionModuleInstance *AuctionsModule

func init() {
	// Register with copyover system
	copyover.RegisterWithVeto("auctions", gatherAuctionState, restoreAuctionState, vetoAuctionCopyover)
}

// SetAuctionModuleInstance stores the module instance for copyover
func SetAuctionModuleInstance(am *AuctionsModule) {
	auctionModuleInstance = am
}

// gatherAuctionState collects auction state before copyover
func gatherAuctionState() (interface{}, error) {
	if auctionModuleInstance == nil {
		mudlog.Info("Copyover", "subsystem", "auctions", "status", "no active instance")
		return nil, nil
	}

	state := AuctionCopyoverState{
		HasActiveAuction: false,
		PastAuctions:     make([]PastAuctionState, 0),
	}

	// Check for active auction
	mgr := &auctionModuleInstance.auctionMgr
	if mgr.ActiveAuction != nil {
		state.HasActiveAuction = true
		state.ActiveAuction = &AuctionItemState{
			ItemId:            mgr.ActiveAuction.ItemData.ItemId,
			ItemName:          mgr.ActiveAuction.ItemData.Name(),
			ItemDescription:   mgr.ActiveAuction.ItemData.GetLongDescription(),
			SellerUserId:      mgr.ActiveAuction.SellerUserId,
			SellerName:        mgr.ActiveAuction.SellerName,
			Anonymous:         mgr.ActiveAuction.Anonymous,
			EndTime:           mgr.ActiveAuction.EndTime,
			MinimumBid:        mgr.ActiveAuction.MinimumBid,
			HighestBid:        mgr.ActiveAuction.HighestBid,
			HighestBidUserId:  mgr.ActiveAuction.HighestBidUserId,
			HighestBidderName: mgr.ActiveAuction.HighestBidderName,
			LastUpdate:        mgr.ActiveAuction.LastUpdate,
		}

		mudlog.Info("Copyover", "subsystem", "auctions",
			"active_auction", state.ActiveAuction.ItemName,
			"bid", state.ActiveAuction.HighestBid,
			"ends", state.ActiveAuction.EndTime)
	}

	// Save past auction history
	for _, past := range mgr.PastAuctions {
		state.PastAuctions = append(state.PastAuctions, PastAuctionState{
			ItemName:   past.ItemName,
			WinningBid: past.WinningBid,
			Anonymous:  past.Anonymous,
			SellerName: past.SellerName,
			BuyerName:  past.BuyerName,
			EndTime:    past.EndTime,
		})
	}

	mudlog.Info("Copyover", "subsystem", "auctions",
		"gathered", "state",
		"active", state.HasActiveAuction,
		"history", len(state.PastAuctions))

	return state, nil
}

// restoreAuctionState restores auction state after copyover
func restoreAuctionState(data interface{}) error {
	if data == nil {
		mudlog.Info("Copyover", "subsystem", "auctions", "status", "no state to restore")
		return nil
	}

	if auctionModuleInstance == nil {
		mudlog.Warn("Copyover", "subsystem", "auctions", "warning", "no module instance to restore to")
		return nil
	}

	// Type assertion with JSON remarshal for safety
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal auction data: %w", err)
	}

	var state AuctionCopyoverState
	if err := json.Unmarshal(jsonData, &state); err != nil {
		return fmt.Errorf("failed to unmarshal auction state: %w", err)
	}

	mgr := &auctionModuleInstance.auctionMgr

	// Restore active auction
	if state.HasActiveAuction && state.ActiveAuction != nil {
		// Recreate the item from its ID
		mgr.ActiveAuction = &AuctionItem{
			ItemData:          items.New(state.ActiveAuction.ItemId),
			SellerUserId:      state.ActiveAuction.SellerUserId,
			SellerName:        state.ActiveAuction.SellerName,
			Anonymous:         state.ActiveAuction.Anonymous,
			EndTime:           state.ActiveAuction.EndTime,
			MinimumBid:        state.ActiveAuction.MinimumBid,
			HighestBid:        state.ActiveAuction.HighestBid,
			HighestBidUserId:  state.ActiveAuction.HighestBidUserId,
			HighestBidderName: state.ActiveAuction.HighestBidderName,
			LastUpdate:        state.ActiveAuction.LastUpdate,
		}

		mudlog.Info("Copyover", "subsystem", "auctions",
			"restored", "active auction",
			"item", state.ActiveAuction.ItemName,
			"bid", state.ActiveAuction.HighestBid)

		// Announce auction restoration to players
		// This could be done via templates or events
	}

	// Restore past auctions
	mgr.PastAuctions = make([]PastAuctionItem, 0, len(state.PastAuctions))
	for _, past := range state.PastAuctions {
		mgr.PastAuctions = append(mgr.PastAuctions, PastAuctionItem{
			ItemName:   past.ItemName,
			WinningBid: past.WinningBid,
			Anonymous:  past.Anonymous,
			SellerName: past.SellerName,
			BuyerName:  past.BuyerName,
			EndTime:    past.EndTime,
		})
	}

	mudlog.Info("Copyover", "subsystem", "auctions",
		"restored", "complete",
		"active", state.HasActiveAuction,
		"history", len(state.PastAuctions))

	return nil
}

// vetoAuctionCopyover checks if it's safe to copyover
func vetoAuctionCopyover() (bool, string) {
	if auctionModuleInstance == nil {
		return true, ""
	}

	mgr := &auctionModuleInstance.auctionMgr

	// Check if auction is about to end
	if mgr.ActiveAuction != nil {
		timeRemaining := time.Until(mgr.ActiveAuction.EndTime)
		if timeRemaining < 30*time.Second && timeRemaining > 0 {
			// Soft veto - warn but allow
			return true, fmt.Sprintf("auction ending in %v", timeRemaining.Round(time.Second))
		}
	}

	return true, ""
}
