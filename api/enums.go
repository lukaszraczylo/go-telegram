package api

// UpdateType identifies an Update payload variant. Used by allowed_updates
// in getUpdates / setWebhook. The Telegram docs do not enumerate these
// values inline (they are derived from the optional fields of Update),
// so the codegen pipeline cannot synthesise this enum and it lives here
// as a hand-curated companion to the generated enums.gen.go.
type UpdateType string

const (
	UpdateMessage                 UpdateType = "message"
	UpdateEditedMessage           UpdateType = "edited_message"
	UpdateChannelPost             UpdateType = "channel_post"
	UpdateEditedChannelPost       UpdateType = "edited_channel_post"
	UpdateBusinessConnection      UpdateType = "business_connection"
	UpdateBusinessMessage         UpdateType = "business_message"
	UpdateEditedBusinessMessage   UpdateType = "edited_business_message"
	UpdateDeletedBusinessMessages UpdateType = "deleted_business_messages"
	UpdateMessageReaction         UpdateType = "message_reaction"
	UpdateMessageReactionCount    UpdateType = "message_reaction_count"
	UpdateInlineQuery             UpdateType = "inline_query"
	UpdateChosenInlineResult      UpdateType = "chosen_inline_result"
	UpdateCallbackQuery           UpdateType = "callback_query"
	UpdateShippingQuery           UpdateType = "shipping_query"
	UpdatePreCheckoutQuery        UpdateType = "pre_checkout_query"
	UpdatePurchasedPaidMedia      UpdateType = "purchased_paid_media"
	UpdatePoll                    UpdateType = "poll"
	UpdatePollAnswer              UpdateType = "poll_answer"
	UpdateMyChatMember            UpdateType = "my_chat_member"
	UpdateChatMember              UpdateType = "chat_member"
	UpdateChatJoinRequest         UpdateType = "chat_join_request"
	UpdateChatBoost               UpdateType = "chat_boost"
	UpdateRemovedChatBoost        UpdateType = "removed_chat_boost"
)
