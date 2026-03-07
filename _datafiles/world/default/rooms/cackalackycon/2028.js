// SAV Lounge - Social Engineering Challenge
// Player must successfully "social engineer" their way to the fragment

const SOCIAL_FRAGMENT = 5102;

// The password changes - it's hidden in the room nouns
const PASSWORD = "open sesame";

function onCommand_pretend(rest, user, room) {
    attemptSocialEngineer(user, room, rest);
    return true;
}

function onCommand_claim(rest, user, room) {
    attemptSocialEngineer(user, room, rest);
    return true;
}

function onCommand_say(rest, user, room) {
    // Check if they're saying the magic phrase
    if (rest.toLowerCase().includes("open sesame")) {
        if (user.HasItemId(SOCIAL_FRAGMENT)) {
            return false; // Let normal say happen
        }
        
        user.SendText("");
        user.SendText("<ansi fg=\"green\">The room falls silent. Someone nods approvingly.</ansi>");
        user.SendText("<ansi fg=\"cyan\">\"Classic. You've passed the test.\"</ansi>");
        user.SendText("");
        
        fragment = CreateItem(SOCIAL_FRAGMENT);
        user.GiveItem(fragment);
        
        user.SendText("<ansi fg=\"lime\">You received: social key fragment</ansi>");
        user.SendText("");
        
        room.SendText(
            "<ansi fg=\"username\">" + user.GetCharacterName(false) + "</ansi> earns the respect of the social engineers.",
            user.UserId()
        );
        return true;
    }
    return false;
}

function attemptSocialEngineer(user, room, rest) {
    if (user.HasItemId(SOCIAL_FRAGMENT)) {
        user.SendText("<ansi fg=\"yellow\">You've already proven yourself here. You have the social fragment.</ansi>");
        return;
    }
    
    user.SendText("");
    user.SendText("<ansi fg=\"yellow\">You try to talk your way to the fragment...</ansi>");
    user.SendText("<ansi fg=\"red\">But everyone here is a social engineer. They see right through you.</ansi>");
    user.SendText("");
    user.SendText("<ansi fg=\"cyan\">Someone whispers: \"Read the room. Literally. The answer is hidden here.\"</ansi>");
    user.SendText("<ansi fg=\"cyan\">\"Look at the props. Then say the magic words.\"</ansi>");
    user.SendText("");
}

function onEnter(user, room) {
    if (user.HasItemId(SOCIAL_FRAGMENT)) {
        return false;
    }
    
    // Hint for new arrivals
    round = UtilGetRoundNumber();
    if (round % 3 == 0) {
        user.SendText("<ansi fg=\"cyan\">A social engineer looks at you appraisingly. \"Looking for something? Examine the props.\"</ansi>");
    }
    return false;
}
