// Phreak Me - Phone phreaking room with FBI Raid chance
// Higher risk area - phreaking attracts federal attention

const REG_DESK_ROOM = 2002;
const RAID_CHANCE_PERCENT = 3;  // Slightly higher - phreaking is illegal!
const CHECK_INTERVAL_ROUNDS = 15;

function onIdle(room) {
    round = UtilGetRoundNumber();
    
    if (round % CHECK_INTERVAL_ROUNDS == 0) {
        if (doFbiRaidCheck(room)) {
            return true;
        }
    }
    
    // Random phone noises
    if (round % 8 == 0) {
        noises = [
            "<ansi fg=\"cyan\">A payphone crackles with distant voices.</ansi>",
            "<ansi fg=\"yellow\">Blue box tones echo through the room.</ansi>",
            "<ansi fg=\"green\">Someone whispers into a red box receiver.</ansi>",
            "<ansi fg=\"cyan\">A dial tone shifts to something... different.</ansi>",
            ""
        ];
        idx = UtilDiceRoll(1, noises.length) - 1;
        if (noises[idx] != "") {
            room.SendText(noises[idx]);
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
    SendRoomMessage(room.RoomId(), "<ansi fg=\"yellow\">The payphones all ring at once - then go dead.</ansi>");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"cyan\">Federal agents burst through every entrance!</ansi>");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"red\">\"TOLL FRAUD INVESTIGATION! NOBODY MOVE!\"</ansi>");
    SendRoomMessage(room.RoomId(), "");
    
    target.SendText("<ansi fg=\"red\">\"Drop the blue box! DROP IT!\"</ansi>");
    target.SendText("<ansi fg=\"yellow\">An agent rips the phone from your hand!</ansi>");
    target.SendText("<ansi fg=\"cyan\">\"We've been tracking these tones for weeks. Gotcha.\"</ansi>");
    target.SendText("");
    
    SendRoomMessage(room.RoomId(), "<ansi fg=\"username\">" + target.GetCharacterName(false) + "</ansi> gets busted mid-phreak!");
    SendRoomMessage(room.RoomId(), "<ansi fg=\"cyan\">Agents escort them out roughly...</ansi>");
    SendRoomMessage(room.RoomId(), "");
    
    target.MoveRoom(REG_DESK_ROOM);
    
    target.SendText("");
    target.SendText("<ansi fg=\"yellow\">You're deposited at registration like yesterday's garbage.</ansi>");
    target.SendText("<ansi fg=\"cyan\">\"Ma Bell sends her regards.\"</ansi>");
    target.SendText("<ansi fg=\"yellow\">The agents vanish, leaving only dial tone.</ansi>");
    target.SendText("");
    
    return true;
}
