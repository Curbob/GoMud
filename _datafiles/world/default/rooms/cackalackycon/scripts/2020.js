// Lockpick Village - FBI Raid zone + Booth Rewards
// Possession of lockpicks is... questionable in some states

const REG_DESK_ROOM = 2002;
const RAID_CHANCE_PERCENT = 2;
const CHECK_INTERVAL_ROUNDS = 15;

// Booth reward tracking
const BOOTHS = [
    { key: "lockpick_booth_2060", gold: 5, name: "Clear Lock" },
    { key: "lockpick_booth_2061", gold: 15, name: "Brass Padlock" },
    { key: "lockpick_booth_2062", gold: 30, name: "Master Lock" },
    { key: "lockpick_booth_2063", gold: 100, name: "Challenge Booth" }
];

function onEnter(user, room) {
    // Check if they just escaped from a booth
    for (var i = 0; i < BOOTHS.length; i++) {
        var booth = BOOTHS[i];
        
        // Did they enter this booth recently?
        if (user.GetTempData(booth.key)) {
            // Clear the temp flag
            user.SetTempData(booth.key, false);
            
            // Have they already completed this booth?
            if (!user.GetMiscData(booth.key + "_completed")) {
                // First time completion!
                user.SetMiscData(booth.key + "_completed", true);
                user.AddGold(booth.gold);
                user.SendText("");
                user.SendText("<ansi fg=\"green-bold\">★ " + booth.name + " COMPLETED! ★</ansi>");
                user.SendText("<ansi fg=\"yellow\">You earned " + booth.gold + " gold!</ansi>");
                user.SendText("");
                
                room.SendText(
                    "<ansi fg=\"username\">" + user.GetCharacterName(false) + "</ansi> emerges from " + booth.name + ", pocketing gold!",
                    user.UserId()
                );
            }
        }
    }
    return false;
}

function onIdle(room) {
    round = UtilGetRoundNumber();
    
    if (round % CHECK_INTERVAL_ROUNDS == 0) {
        if (doFbiRaidCheck(room)) {
            return true;
        }
    }
    
    // Ambient lockpicking sounds
    if (round % 7 == 0) {
        sounds = [
            "<ansi fg=\"cyan\">*click* Someone pops a padlock open.</ansi>",
            "<ansi fg=\"yellow\">Tension wrenches scrape against pin stacks.</ansi>",
            "<ansi fg=\"green\">A handcuff clicks open to scattered applause.</ansi>",
            "<ansi fg=\"cyan\">\"That's a false set, feel the difference?\"</ansi>",
            ""
        ];
        idx = UtilDiceRoll(1, sounds.length) - 1;
        if (sounds[idx] != "") {
            room.SendText(sounds[idx]);
            return true;
        }
    }
    
    return false;
}

function doFbiRaidCheck(room) {
    var players = room.GetPlayers();
    if (players.length == 0) {
        return false;
    }
    
    if (UtilDiceRoll(1, 100) > RAID_CHANCE_PERCENT) {
        return false;
    }
    
    var target = null;
    for (var i = 0; i < players.length; i++) {
        var player = players[i];
        if (player.GetRace().toLowerCase() == "fbi agent") {
            continue;
        }
        target = player;
        break;
    }
    
    if (target == null) {
        return false;
    }
    
    SendRoomMessage(room.RoomId(), "");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"red-bold\">═══════════════════════════════════════════</ansi>");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"red-bold\">           🚨 FBI RAID! 🚨</ansi>");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"red-bold\">═══════════════════════════════════════════</ansi>");
    SendRoomMessage(room.RoomId(), "");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"yellow\">Every lock in the room clicks shut simultaneously!</ansi>");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"cyan\">Federal agents pour in from every entrance!</ansi>");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"red\">\"POSSESSION OF BURGLARY TOOLS! HANDS UP!\"</ansi>");
    SendRoomMessage(room.RoomId(), "");
    
    target.SendText("<ansi fg=\"red\">\"DROP THE PICKS! NOW!\"</ansi>");
    target.SendText("<ansi fg=\"yellow\">An agent slaps the tension wrench out of your hand!</ansi>");
    target.SendText("<ansi fg=\"cyan\">\"Funny how you're so good at locks but couldn't see us coming.\"</ansi>");
    target.SendText("");
    
    SendRoomMessage(room.RoomId(), "<ansi fg=\"username\">" + target.GetCharacterName(false) + "</ansi> gets caught with picks in hand!");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"cyan\">Agents drag them away in cuffs... ironic.</ansi>");
    SendRoomMessage(room.RoomId(), "");
    
    target.MoveRoom(REG_DESK_ROOM);
    
    target.SendText("");
    target.SendText("<ansi fg=\"yellow\">You're dumped at registration, still flexing your wrists.</ansi>");
    target.SendText("<ansi fg=\"cyan\">\"Those picks are evidence now. Good luck at the village.\"</ansi>");
    target.SendText("<ansi fg=\"yellow\">The agents disappear, leaving you pick-less.</ansi>");
    target.SendText("");
    
    return true;
}
