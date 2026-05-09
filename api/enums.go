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

// DiceEmoji is the set of emoji values accepted by sendDice. Telegram's
// canonical list is "🎲", "🎯", "🏀", "⚽", "🎳", "🎰". The codegen
// scraper drops these values during regex extraction (multi-byte
// boundary issues with curly-quoted emoji), so this enum is hand-
// curated and wired into SendDiceParams.Emoji via the per-field type
// override in cmd/genapi/emitter.go.
type DiceEmoji string

const (
	DiceEmojiDice        DiceEmoji = "🎲"
	DiceEmojiDart        DiceEmoji = "🎯"
	DiceEmojiBasketball  DiceEmoji = "🏀"
	DiceEmojiFootball    DiceEmoji = "⚽"
	DiceEmojiBowling     DiceEmoji = "🎳"
	DiceEmojiSlotMachine DiceEmoji = "🎰"
)

// ReactionEmoji is the set of emoji Telegram allows in a
// ReactionTypeEmoji.Emoji value. Hand-curated from
// https://core.telegram.org/bots/api#reactiontypeemoji because the
// scraper's curly-quote regex strips the emoji literals (byte-boundary
// issue on multi-byte sequences). Names mirror the Unicode CLDR short
// name where one exists; otherwise a stable common-English label.
// Telegram occasionally extends this set — passers of unrecognised
// strings still type-check (ReactionEmoji is a string alias) so this
// list need not block runtime use of newer values.
type ReactionEmoji string

const (
	ReactionEmojiHeart                     ReactionEmoji = "❤"
	ReactionEmojiThumbsUp                  ReactionEmoji = "👍"
	ReactionEmojiThumbsDown                ReactionEmoji = "👎"
	ReactionEmojiFire                      ReactionEmoji = "🔥"
	ReactionEmojiSmilingFaceWithHearts     ReactionEmoji = "🥰"
	ReactionEmojiClappingHands             ReactionEmoji = "👏"
	ReactionEmojiBeamingFace               ReactionEmoji = "😁"
	ReactionEmojiThinkingFace              ReactionEmoji = "🤔"
	ReactionEmojiExplodingHead             ReactionEmoji = "🤯"
	ReactionEmojiScreamingFace             ReactionEmoji = "😱"
	ReactionEmojiCursingFace               ReactionEmoji = "🤬"
	ReactionEmojiCryingFace                ReactionEmoji = "😢"
	ReactionEmojiPartyPopper               ReactionEmoji = "🎉"
	ReactionEmojiStarStruck                ReactionEmoji = "🤩"
	ReactionEmojiVomiting                  ReactionEmoji = "🤮"
	ReactionEmojiPileOfPoo                 ReactionEmoji = "💩"
	ReactionEmojiFoldedHands               ReactionEmoji = "🙏"
	ReactionEmojiOKHand                    ReactionEmoji = "👌"
	ReactionEmojiDove                      ReactionEmoji = "🕊"
	ReactionEmojiClown                     ReactionEmoji = "🤡"
	ReactionEmojiYawning                   ReactionEmoji = "🥱"
	ReactionEmojiWoozyFace                 ReactionEmoji = "🥴"
	ReactionEmojiHeartEyes                 ReactionEmoji = "😍"
	ReactionEmojiWhale                     ReactionEmoji = "🐳"
	ReactionEmojiHeartOnFire               ReactionEmoji = "❤‍🔥"
	ReactionEmojiNewMoonFace               ReactionEmoji = "🌚"
	ReactionEmojiHotDog                    ReactionEmoji = "🌭"
	ReactionEmojiHundredPoints             ReactionEmoji = "💯"
	ReactionEmojiRollingOnFloor            ReactionEmoji = "🤣"
	ReactionEmojiLightning                 ReactionEmoji = "⚡"
	ReactionEmojiBanana                    ReactionEmoji = "🍌"
	ReactionEmojiTrophy                    ReactionEmoji = "🏆"
	ReactionEmojiBrokenHeart               ReactionEmoji = "💔"
	ReactionEmojiRaisedEyebrow             ReactionEmoji = "🤨"
	ReactionEmojiNeutralFace               ReactionEmoji = "😐"
	ReactionEmojiStrawberry                ReactionEmoji = "🍓"
	ReactionEmojiChampagne                 ReactionEmoji = "🍾"
	ReactionEmojiKissMark                  ReactionEmoji = "💋"
	ReactionEmojiMiddleFinger              ReactionEmoji = "🖕"
	ReactionEmojiDevil                     ReactionEmoji = "😈"
	ReactionEmojiSleeping                  ReactionEmoji = "😴"
	ReactionEmojiLoudlyCrying              ReactionEmoji = "😭"
	ReactionEmojiNerd                      ReactionEmoji = "🤓"
	ReactionEmojiGhost                     ReactionEmoji = "👻"
	ReactionEmojiManTechnologist           ReactionEmoji = "👨‍💻"
	ReactionEmojiEyes                      ReactionEmoji = "👀"
	ReactionEmojiJackOLantern              ReactionEmoji = "🎃"
	ReactionEmojiSeeNoEvil                 ReactionEmoji = "🙈"
	ReactionEmojiHalo                      ReactionEmoji = "😇"
	ReactionEmojiFearful                   ReactionEmoji = "😨"
	ReactionEmojiHandshake                 ReactionEmoji = "🤝"
	ReactionEmojiWriting                   ReactionEmoji = "✍"
	ReactionEmojiHugging                   ReactionEmoji = "🤗"
	ReactionEmojiSaluting                  ReactionEmoji = "🫡"
	ReactionEmojiSantaClaus                ReactionEmoji = "🎅"
	ReactionEmojiChristmasTree             ReactionEmoji = "🎄"
	ReactionEmojiSnowman                   ReactionEmoji = "☃"
	ReactionEmojiNailPolish                ReactionEmoji = "💅"
	ReactionEmojiZanyFace                  ReactionEmoji = "🤪"
	ReactionEmojiMoai                      ReactionEmoji = "🗿"
	ReactionEmojiCool                      ReactionEmoji = "🆒"
	ReactionEmojiHeartWithArrow            ReactionEmoji = "💘"
	ReactionEmojiHearNoEvil                ReactionEmoji = "🙉"
	ReactionEmojiUnicorn                   ReactionEmoji = "🦄"
	ReactionEmojiKissingFace               ReactionEmoji = "😘"
	ReactionEmojiPill                      ReactionEmoji = "💊"
	ReactionEmojiSpeakNoEvil               ReactionEmoji = "🙊"
	ReactionEmojiSmilingFaceWithSunglasses ReactionEmoji = "😎"
	ReactionEmojiAlienMonster              ReactionEmoji = "👾"
	ReactionEmojiManShrugging              ReactionEmoji = "🤷‍♂"
	ReactionEmojiPersonShrugging           ReactionEmoji = "🤷"
	ReactionEmojiWomanShrugging            ReactionEmoji = "🤷‍♀"
	ReactionEmojiPoutingFace               ReactionEmoji = "😡"
)
