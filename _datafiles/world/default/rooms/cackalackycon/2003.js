// Con Floor - Secret exit discovery script
// Players can find the service exit by looking at or searching the door

function referencesServiceDoor(rest) {
    const text = (rest || "").toLowerCase();
    return text.includes("door") || text.includes("service");
}

function onCommand_look(rest, user, room) {
    if (referencesServiceDoor(rest)) {
        revealServiceExit(user, room);
        return true;
    }
    return false;
}

function onCommand_examine(rest, user, room) {
    if (referencesServiceDoor(rest)) {
        revealServiceExit(user, room);
        return true;
    }
    return false;
}

function onCommand_search(rest, user, room) {
    if (referencesServiceDoor(rest)) {
        revealServiceExit(user, room);
        return true;
    }
    return false;
}

function revealServiceExit(user, room) {
    user.SendText("");
    user.SendText("<ansi fg=\"yellow\">You push aside the banner and try the door marked 'SERVICE'...</ansi>");
    user.SendText("<ansi fg=\"green\">It's unlocked! The door swings open, revealing a dim corridor.</ansi>");
    user.SendText("");
    user.SendText("You discovered a secret exit: <ansi fg=\"secret-exit\">service</ansi>");
    user.SendText("");
    
    room.SendText(
        "<ansi fg=\"username\">" + user.GetCharacterName(false) + "</ansi> pushes aside a banner and slips through a service door!",
        user.UserId()
    );
    
    // Move them through
    user.MoveRoom(2050);
}
