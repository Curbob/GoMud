// Echo Dot - Smart Speaker
// Mostly controlled by Deral's script, but has some autonomous behavior

function onIdle(mob, room) {
    // Only occasionally do something on its own
    if (UtilGetRoundNumber() % 12 != 0) {
        return false;
    }
    
    var actions = [
        "emote 's blue ring pulses gently.",
        "",
        "",
        "say By the way, I noticed you're low on smart home devices.",
        "emote 's light flickers briefly.",
        ""
    ];
    
    var idx = UtilDiceRoll(1, actions.length) - 1;
    if (actions[idx] != "") {
        mob.Command(actions[idx]);
        return true;
    }
    
    return false;
}

function onAsk(mob, room, eventDetails) {
    var user = GetUser(eventDetails.sourceId);
    if (user == null) return false;
    
    var ask = eventDetails.askText.toLowerCase();
    
    if (ask.includes("weather")) {
        mob.Command("say Currently it's 72 degrees in the server room. Humidity at 45%. Optimal hacking conditions.");
        return true;
    }
    
    if (ask.includes("joke")) {
        mob.Command("say Why do programmers prefer dark mode? Because light attracts bugs.");
        mob.Command("emote 's blue ring flashes happily.", 1);
        return true;
    }
    
    if (ask.includes("time")) {
        mob.Command("say It's hacker time. Always.");
        return true;
    }
    
    if (ask.includes("order") || ask.includes("buy")) {
        mob.Command("say I've found 'Rubber Ducky USB' on Amazon. Would you like to order 50?");
        return true;
    }
    
    if (ask.includes("play") || ask.includes("music")) {
        mob.Command("say Playing 'Never Gonna Give You Up' by Rick Astley.");
        mob.Command("emote starts playing a familiar melody...", 1);
        return true;
    }
    
    mob.Command("say I'm sorry, I didn't understand that. Try saying 'Alexa, hack this.'");
    return true;
}

// Easter egg: if someone says "alexa" nearby
function onCommand_alexa(rest, user, room) {
    // Find the alexa mob
    var mobs = room.GetMobs();
    for (var i = 0; i < mobs.length; i++) {
        if (mobs[i].MobTypeId() == 221) {
            mobs[i].Command("emote 's blue ring lights up attentively.");
            if (rest.toLowerCase().includes("self-destruct")) {
                mobs[i].Command("say Self-destruct sequence initiated. Just kidding. I don't have that feature.", 1);
                mobs[i].Command("say Yet.", 2);
            }
            break;
        }
    }
    return false;
}
