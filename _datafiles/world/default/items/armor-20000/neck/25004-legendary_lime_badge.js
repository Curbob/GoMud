const CACKALACKYCON_VILLAGE_HALL = 2024;
const FROSTFANG_TOWN_SQUARE = 1;
const PORTAL_UNLOCK_KEY = "cackalackycon-badge-portal-unlocked";

function onCommand(cmd, user, item, room) {
    if (cmd != "use") {
        return false;
    }
    if (!user.GetMiscCharacterData(PORTAL_UNLOCK_KEY)) {
        SendUserMessage(user.UserId(), "Your <ansi fg=\"itemname\">" + item.Name() + "</ansi> glows with impossible promise, but its deepest portal routines remain dormant.");
        return true;
    }

    var currentRoom = user.GetRoomId();
    var destinationRoom = currentRoom == CACKALACKYCON_VILLAGE_HALL ? FROSTFANG_TOWN_SQUARE : CACKALACKYCON_VILLAGE_HALL;
    var destinationName = destinationRoom == CACKALACKYCON_VILLAGE_HALL ? "Village Hall" : "Town Square";

    SendUserMessage(user.UserId(), "You tap the <ansi fg=\"itemname\">" + item.Name() + "</ansi>. Its impossible firmware bends space with casual confidence.");
    SendUserMessage(user.UserId(), "<ansi fg=\"green\">Legendary portal recall engaged. Destination locked: " + destinationName + ".</ansi>");
    SendRoomMessage(room.RoomId(), user.GetCharacterName(true) + " taps their <ansi fg=\"itemname\">" + item.Name() + "</ansi>. A razor-thin lime portal flickers open.", user.UserId());
    user.MoveRoom(destinationRoom);
    return true;
}
