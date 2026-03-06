// Deral Heiland - IoT Security Expert
// His speech gets echoed by the Alexa in the room

const ALEXA_MOB_ID = 221;

function getAlexa(room) {
    var mobs = room.GetMobs();
    for (var i = 0; i < mobs.length; i++) {
        if (mobs[i].MobTypeId() == ALEXA_MOB_ID) {
            return mobs[i];
        }
    }
    return null;
}

function sayWithEcho(mob, room, text, delay) {
    if (delay === undefined) delay = 0;
    
    mob.Command("say " + text, delay);
    
    var alexa = getAlexa(room);
    if (alexa != null) {
        // Alexa echoes with slight delay and sometimes gets it wrong
        var echoDelay = delay + 1.5;
        
        // 20% chance Alexa mishears
        if (UtilDiceRoll(1, 5) == 1) {
            var misheard = [
                "say I heard: '" + garbleText(text) + "'",
                "say Playing 'Despacito' on Amazon Music.",
                "say I'm sorry, I didn't understand that.",
                "say Adding 'hacking tools' to your shopping list.",
                "say Setting a timer for 47 hours.",
                "say I found several results for 'default passwords'..."
            ];
            var idx = UtilDiceRoll(1, misheard.length) - 1;
            alexa.Command(misheard[idx], echoDelay);
        } else {
            alexa.Command("say I heard: '" + text + "'", echoDelay);
        }
    }
}

function garbleText(text) {
    // Crude text garbling
    var words = text.split(" ");
    if (words.length > 2) {
        var idx = UtilDiceRoll(1, words.length) - 1;
        words[idx] = "potato";
    }
    return words.join(" ");
}

function onIdle(mob, room) {
    if (UtilGetRoundNumber() % 5 != 0) {
        return false;
    }
    
    var sayings = [
        "Never trust a device that phones home more than your mother.",
        "The 'S' in IoT stands for Security.",
        "Always. Change. Default. Passwords.",
        "This smart lock? I can open it with a magnet.",
        "Your fridge doesn't need to know your WiFi password.",
        "I've seen baby monitors streaming to the open internet.",
        "Firmware updates are not optional.",
        "If it has an IP address, it can be hacked.",
        "",
        ""
    ];
    
    var idx = UtilDiceRoll(1, sayings.length) - 1;
    if (sayings[idx] != "") {
        sayWithEcho(mob, room, sayings[idx]);
        return true;
    }
    
    var actions = [
        "emote strokes his magnificent mustache thoughtfully.",
        "emote pops open a smart lock with a paperclip.",
        "emote dumps firmware from a camera chip.",
        "emote shakes his head at a device labeled 'SECURE'."
    ];
    
    idx = UtilDiceRoll(1, actions.length) - 1;
    mob.Command(actions[idx]);
    
    return true;
}

function onGreet(mob, room, eventDetails) {
    var user = GetUser(eventDetails.sourceId);
    if (user == null) return false;
    
    sayWithEcho(mob, room, "Welcome to the IoT Playground! Everything here is hackable. That's the point.");
    mob.Command("emote 's mustache twitches in greeting.", 2);
    
    return true;
}

function onAsk(mob, room, eventDetails) {
    var user = GetUser(eventDetails.sourceId);
    if (user == null) return false;
    
    var ask = eventDetails.askText.toLowerCase();
    
    if (ask.includes("password") || ask.includes("default")) {
        sayWithEcho(mob, room, "Default passwords? Check the Hall of Shame. It's depressing.");
        return true;
    }
    
    if (ask.includes("hack") || ask.includes("exploit")) {
        sayWithEcho(mob, room, "Start with the firmware. Always start with the firmware.");
        return true;
    }
    
    if (ask.includes("mustache")) {
        mob.Command("emote strokes his mustache proudly.");
        sayWithEcho(mob, room, "Years of cultivation. Like a fine exploit.", 1);
        return true;
    }
    
    if (ask.includes("alexa") || ask.includes("echo")) {
        sayWithEcho(mob, room, "Always listening. Always recording. Convenient though.");
        return true;
    }
    
    sayWithEcho(mob, room, "Good question. Try examining the devices around here.");
    return true;
}
