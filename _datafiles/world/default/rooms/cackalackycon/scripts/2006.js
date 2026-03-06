// LobbyCon - The Hallway Track
// Features a "Visitor Board" showing everyone who has passed through

function onEnter(user, room) {
    // Add this user to the visitor list
    var visitors = room.GetLongTermData("visitors");
    if (visitors == null) {
        visitors = {};
    }
    
    var name = user.GetCharacterName(false);
    var now = UtilGetRoundNumber();
    
    // Track name and last visit
    visitors[name] = now;
    room.SetLongTermData("visitors", visitors);
    
    // Occasionally mention the board
    if (now % 5 == 0) {
        user.SendText("<ansi fg=\"cyan\">A big board on the wall catches your eye. You could <ansi fg=\"command\">examine board</ansi> to see who's been through LobbyCon.</ansi>");
    }
    
    return false;
}

function onCommand_examine(rest, user, room) {
    if (rest.toLowerCase().includes("board") || 
        rest.toLowerCase().includes("sign") ||
        rest.toLowerCase().includes("names")) {
        showBoard(user, room);
        return true;
    }
    return false;
}

function onCommand_read(rest, user, room) {
    if (rest.toLowerCase().includes("board") || 
        rest.toLowerCase().includes("sign") ||
        rest.toLowerCase().includes("names")) {
        showBoard(user, room);
        return true;
    }
    return false;
}

function onCommand_look(rest, user, room) {
    if (rest.toLowerCase().includes("board") || 
        rest.toLowerCase().includes("sign")) {
        showBoard(user, room);
        return true;
    }
    return false;
}

function showBoard(user, room) {
    var visitors = room.GetLongTermData("visitors");
    
    user.SendText("");
    user.SendText("<ansi fg=\"cyan-bold\">═══════════════════════════════════════════</ansi>");
    user.SendText("<ansi fg=\"yellow-bold\">    LOBBYCON VISITOR BOARD</ansi>");
    user.SendText("<ansi fg=\"white\">    \"I was here!\"</ansi>");
    user.SendText("<ansi fg=\"cyan-bold\">═══════════════════════════════════════════</ansi>");
    user.SendText("");
    
    if (visitors == null || Object.keys(visitors).length == 0) {
        user.SendText("<ansi fg=\"white\">The board is empty. Be the first to sign!</ansi>");
    } else {
        var names = Object.keys(visitors);
        
        // Sort by most recent
        names.sort(function(a, b) {
            return visitors[b] - visitors[a];
        });
        
        // Show up to 20 most recent
        var shown = 0;
        for (var i = 0; i < names.length && shown < 20; i++) {
            user.SendText("  <ansi fg=\"username\">" + names[i] + "</ansi>");
            shown++;
        }
        
        if (names.length > 20) {
            user.SendText("  <ansi fg=\"white\">...and " + (names.length - 20) + " more hackers</ansi>");
        }
        
        user.SendText("");
        user.SendText("<ansi fg=\"yellow\">" + names.length + " hackers have passed through LobbyCon</ansi>");
    }
    
    user.SendText("<ansi fg=\"cyan-bold\">═══════════════════════════════════════════</ansi>");
    user.SendText("");
}

// Add board to room nouns reference
function onCommand_sign(rest, user, room) {
    user.SendText("<ansi fg=\"cyan\">Your name was added to the board when you arrived. <ansi fg=\"command\">Examine board</ansi> to see it!</ansi>");
    return true;
}
