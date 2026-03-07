function onEnter(user, room) {
    var visitors = room.GetLongTermData("visitors");
    if (visitors == null) {
        visitors = {};
    }
    visitors[user.GetCharacterName(false)] = UtilGetRoundNumber();
    room.SetLongTermData("visitors", visitors);
    return false;
}

function onCommand(cmd, rest, user, room) {
    if (cmd == "board") {
        var visitors = room.GetLongTermData("visitors");
        user.SendText("===========================================");
        user.SendText("    LOBBYCON VISITOR BOARD");
        user.SendText("===========================================");
        if (visitors == null || Object.keys(visitors).length == 0) {
            user.SendText("  (empty)");
        } else {
            var names = Object.keys(visitors);
            for (var i = 0; i < names.length && i < 20; i++) {
                user.SendText("  " + names[i]);
            }
            if (names.length > 20) {
                user.SendText("  ...and " + (names.length - 20) + " more");
            }
        }
        user.SendText("===========================================");
        return true;
    }
    return false;
}
