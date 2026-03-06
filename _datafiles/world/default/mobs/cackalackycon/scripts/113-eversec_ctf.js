// Eversec CTF - Three hackers that move as one
// They finish each other's sentences and speak in rotation

var speakerIndex = 0;
const SPEAKERS = ["Ev", "Er", "Sec"];

function speak(mob, text) {
    var speaker = SPEAKERS[speakerIndex];
    speakerIndex = (speakerIndex + 1) % 3;
    mob.Command("say " + speaker + ": " + text);
}

function allSpeak(mob, text1, text2, text3) {
    mob.Command("say Ev: " + text1);
    mob.Command("say Er: " + text2, 0.8);
    mob.Command("say Sec: " + text3, 1.6);
}

function onIdle(mob, room) {
    if (UtilGetRoundNumber() % 5 != 0) {
        return false;
    }
    
    var roll = UtilDiceRoll(1, 10);
    
    if (roll <= 3) {
        // Synchronized actions
        var actions = [
            "emote all three type the exact same command simultaneously.",
            "emote pivot in unison to check the scoreboard.",
            "emote nod at each other, communicating without words.",
            "emote huddle closer, becoming an even tighter formation.",
            "emote crack their knuckles in perfect sync. All six hands."
        ];
        var idx = UtilDiceRoll(1, actions.length) - 1;
        mob.Command(actions[idx]);
        return true;
    } 
    else if (roll <= 6) {
        // Finish each other's sentences
        var sentences = [
            ["The flag is...", "...probably in...", "...the environment variables."],
            ["This binary...", "...has no...", "...stack canary."],
            ["The web challenge...", "...is definitely...", "...SQL injection."],
            ["Someone just...", "...submitted a...", "...wrong flag. Again."],
            ["The crypto...", "...uses weak...", "...random number generation."],
            ["We should add...", "...more troll...", "...challenges next year."]
        ];
        var idx = UtilDiceRoll(1, sentences.length) - 1;
        allSpeak(mob, sentences[idx][0], sentences[idx][1], sentences[idx][2]);
        return true;
    }
    else if (roll <= 8) {
        // Single speaker comments
        var comments = [
            "Another solve on web100.",
            "Scoreboard's getting tight.",
            "Who keeps submitting 'flag{test}'?",
            "The reversing challenge is brutal this year.",
            "Someone's close to first blood on pwn500.",
            "We see everything. We know everything."
        ];
        var idx = UtilDiceRoll(1, comments.length) - 1;
        speak(mob, comments[idx]);
        return true;
    }
    
    return false;
}

function onGreet(mob, room, eventDetails) {
    var user = GetUser(eventDetails.sourceId);
    if (user == null) return false;
    
    allSpeak(mob, 
        "Welcome...", 
        "...to the...", 
        "...CTF Arena."
    );
    mob.Command("emote all three turn to face " + user.GetCharacterName(false) + " simultaneously.", 2);
    mob.Command("say Ev: Your ranking is...", 3);
    mob.Command("say Er: ...currently...", 3.5);
    mob.Command("say Sec: ...unimpressive.", 4);
    
    return true;
}

function onAsk(mob, room, eventDetails) {
    var user = GetUser(eventDetails.sourceId);
    if (user == null) return false;
    
    var ask = eventDetails.askText.toLowerCase();
    
    if (ask.includes("hint") || ask.includes("help")) {
        allSpeak(mob, 
            "Hints?",
            "We don't...",
            "...do hints."
        );
        mob.Command("emote all three smile identically. It's unsettling.", 2);
        return true;
    }
    
    if (ask.includes("flag") || ask.includes("ctf")) {
        allSpeak(mob,
            "The flags...",
            "...are hidden...",
            "...where they should be."
        );
        return true;
    }
    
    if (ask.includes("name") || ask.includes("three") || ask.includes("who")) {
        mob.Command("say Ev: I am Ev.");
        mob.Command("say Er: I am Er.", 0.5);
        mob.Command("say Sec: I am Sec.", 1);
        mob.Command("say Ev: Together...", 1.8);
        mob.Command("say Er: ...we are...", 2.3);
        mob.Command("say Sec: ...Eversec.", 2.8);
        mob.Command("emote speak in perfect unison: \"Always have been.\"", 3.5);
        return true;
    }
    
    if (ask.includes("score") || ask.includes("rank") || ask.includes("win")) {
        allSpeak(mob,
            "Check the...",
            "...scoreboard...",
            "...like everyone else."
        );
        return true;
    }
    
    // Default - cryptic response
    allSpeak(mob,
        "Interesting...",
        "...question...",
        "...next."
    );
    return true;
}
