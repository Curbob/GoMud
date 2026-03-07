// Service Hallway - FBI Raid zone
// Sketchy back-of-house area

const REG_DESK_ROOM = 2002;
const RAID_CHANCE_PERCENT = 2;
const CHECK_INTERVAL_ROUNDS = 15;

function onIdle(room) {
    round = UtilGetRoundNumber();
    
    if (round % CHECK_INTERVAL_ROUNDS == 0) {
        if (doFbiRaidCheck(room)) {
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
    SendRoomMessage(room.RoomId(), "<ansi fg=\"yellow\">Flashlights cut through the dim hallway!</ansi>");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"cyan\">\"Nobody move! Federal agents!\"</ansi>");
    SendRoomMessage(room.RoomId(), "");
    
    target.SendText("<ansi fg=\"red\">\"YOU! HANDS WHERE I CAN SEE THEM!\"</ansi>");
    target.SendText("<ansi fg=\"yellow\">Before you can react, you're face-down on the cold concrete!</ansi>");
    target.SendText("<ansi fg=\"cyan\">\"Got one! Unauthorized access to restricted areas!\"</ansi>");
    target.SendText("");
    
    SendRoomMessage(room.RoomId(), "<ansi fg=\"username\">" + target.GetCharacterName(false) + "</ansi> gets slammed against the wall by agents!");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"cyan\">The feds drag them away...</ansi>");
    SendRoomMessage(room.RoomId(), "");
    
    target.MoveRoom(REG_DESK_ROOM);
    
    target.SendText("");
    target.SendText("<ansi fg=\"yellow\">The agents dump you at the registration desk.</ansi>");
    target.SendText("<ansi fg=\"cyan\">\"Stick to the public areas, genius.\"</ansi>");
    target.SendText("<ansi fg=\"yellow\">They disappear back into the shadows.</ansi>");
    target.SendText("");
    
    return true;
}
