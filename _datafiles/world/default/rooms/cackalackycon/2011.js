// CTF Arena - Flag Submission Challenge
// Players must find 3 flags hidden around the con and submit them here

const CTF_FRAGMENT = 5103;

// Flags hidden in room descriptions/nouns around the con
// Players examine things and find these strings
const FLAGS = {
    "flag{l1m3_t1m3}": "web",           // Hidden in bar nouns
    "flag{h4ck_th3_pl4n3t}": "misc",    // Hidden in main talk track
    "flag{s0c14l_3ng1n33r}": "social"   // Hidden in SAV lounge
};

const FLAGS_NEEDED = 2; // Need 2 of 3 to get fragment

function onCommand_submit(rest, user, room) {
    if (user.HasItemId(CTF_FRAGMENT)) {
        user.SendText("<ansi fg=\"yellow\">You've already earned the CTF fragment. Good work, hacker.</ansi>");
        return true;
    }
    
    var flag = rest.toLowerCase().trim();
    
    // Check if it's a valid flag
    var category = null;
    for (var f in FLAGS) {
        if (flag == f.toLowerCase()) {
            category = FLAGS[f];
            break;
        }
    }
    
    if (category == null) {
        user.SendText("");
        user.SendText("<ansi fg=\"red\">INVALID FLAG</ansi>");
        user.SendText("<ansi fg=\"yellow\">Uncue glances at you. \"That's not a flag. Keep hunting.\"</ansi>");
        user.SendText("");
        return true;
    }
    
    // Track submitted flags in MiscData
    var submitted = user.GetMiscData("ctf_flags");
    if (submitted == null) {
        submitted = [];
    }
    
    // Check if already submitted
    for (var i = 0; i < submitted.length; i++) {
        if (submitted[i] == flag) {
            user.SendText("");
            user.SendText("<ansi fg=\"yellow\">DUPLICATE FLAG</ansi>");
            user.SendText("<ansi fg=\"cyan\">\"Already got that one. Find something new.\"</ansi>");
            user.SendText("");
            return true;
        }
    }
    
    // Add the flag
    submitted.push(flag);
    user.SetMiscData("ctf_flags", submitted);
    
    user.SendText("");
    user.SendText("<ansi fg=\"green\">✓ FLAG ACCEPTED!</ansi>");
    user.SendText("<ansi fg=\"cyan\">Category: " + category.toUpperCase() + "</ansi>");
    user.SendText("<ansi fg=\"yellow\">Flags submitted: " + submitted.length + "/" + FLAGS_NEEDED + "</ansi>");
    user.SendText("");
    
    room.SendText(
        "<ansi fg=\"username\">" + user.GetCharacterName(false) + "</ansi> submits a flag! The scoreboard updates.",
        user.UserId()
    );
    
    // Check if they've submitted enough
    if (submitted.length >= FLAGS_NEEDED) {
        user.SendText("<ansi fg=\"green-bold\">════════════════════════════════════════</ansi>");
        user.SendText("<ansi fg=\"lime\">       CHALLENGE COMPLETE!</ansi>");
        user.SendText("<ansi fg=\"green-bold\">════════════════════════════════════════</ansi>");
        user.SendText("");
        user.SendText("<ansi fg=\"cyan\">Uncue nods approvingly.</ansi>");
        user.SendText("<ansi fg=\"white\">\"Not bad. You've earned this.\"</ansi>");
        user.SendText("");
        
        var fragment = CreateItem(CTF_FRAGMENT);
        user.GiveItem(fragment);
        
        user.SendText("<ansi fg=\"lime\">You received: CTF key fragment</ansi>");
        user.SendText("");
        
        room.SendText(
            "<ansi fg=\"username\">" + user.GetCharacterName(false) + "</ansi> earns a key fragment from the CTF masters!",
            user.UserId()
        );
    }
    
    return true;
}

function onCommand_flags(rest, user, room) {
    showStatus(user);
    return true;
}

function onCommand_score(rest, user, room) {
    showStatus(user);
    return true;
}

function onCommand_status(rest, user, room) {
    showStatus(user);
    return true;
}

function showStatus(user) {
    var submitted = user.GetMiscData("ctf_flags");
    var count = 0;
    if (submitted != null) {
        count = submitted.length;
    }
    
    user.SendText("");
    user.SendText("<ansi fg=\"cyan\">═══════════════════════════════════════════</ansi>");
    user.SendText("<ansi fg=\"yellow\">        YOUR CTF STATUS</ansi>");
    user.SendText("<ansi fg=\"cyan\">═══════════════════════════════════════════</ansi>");
    user.SendText("");
    user.SendText("<ansi fg=\"white\">Flags submitted: " + count + "/" + FLAGS_NEEDED + "</ansi>");
    
    if (user.HasItemId(CTF_FRAGMENT)) {
        user.SendText("<ansi fg=\"green\">Fragment: OBTAINED ✓</ansi>");
    } else {
        user.SendText("<ansi fg=\"yellow\">Fragment: Not yet earned</ansi>");
    }
    
    user.SendText("");
    user.SendText("<ansi fg=\"white\">Flags are hidden around the con.</ansi>");
    user.SendText("<ansi fg=\"white\">Examine things. Look carefully.</ansi>");
    user.SendText("<ansi fg=\"white\">Format: submit flag{...}</ansi>");
    user.SendText("<ansi fg=\"cyan\">═══════════════════════════════════════════</ansi>");
    user.SendText("");
}

function onEnter(user, room) {
    if (user.HasItemId(CTF_FRAGMENT)) {
        return false;
    }
    
    // Hint for new arrivals
    var round = UtilGetRoundNumber();
    if (round % 4 == 0) {
        user.SendText("<ansi fg=\"cyan\">Uncue looks up. \"Here to compete? Find flags hidden around the con. Submit them here.\"</ansi>");
    }
    return false;
}
