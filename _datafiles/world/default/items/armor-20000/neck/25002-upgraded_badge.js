const CACKALACKYCON_VILLAGE_HALL = 2024;
const FROSTFANG_TOWN_SQUARE = 1;
const PORTAL_UNLOCK_KEY = "cackalackycon-badge-portal-unlocked";

function onCommand(cmd, user, item, room) {
    if (cmd != "use") {
        return false;
    }
    if (!user.GetMiscCharacterData(PORTAL_UNLOCK_KEY)) {
        SendUserMessage(user.UserId(), "Your <ansi fg=\"itemname\">" + item.Name() + "</ansi> hums softly, but its strange features haven't unlocked yet.");
        return true;
    }

    var currentRoom = user.GetRoomId();
    var destinationRoom = currentRoom == CACKALACKYCON_VILLAGE_HALL ? FROSTFANG_TOWN_SQUARE : CACKALACKYCON_VILLAGE_HALL;
    var destinationName = destinationRoom == CACKALACKYCON_VILLAGE_HALL ? "Village Hall" : "Town Square";

    SendUserMessage(user.UserId(), "You tap the <ansi fg=\"itemname\">" + item.Name() + "</ansi>. Hidden routines unfurl across the tiny screen.");
    SendUserMessage(user.UserId(), "<ansi fg=\"green\">Portal recall engaged. Destination locked: " + destinationName + ".</ansi>");
    SendRoomMessage(room.RoomId(), user.GetCharacterName(true) + " taps their <ansi fg=\"itemname\">" + item.Name() + "</ansi>. Lime light flickers around them.", user.UserId());
    user.MoveRoom(destinationRoom);
    return true;
}
