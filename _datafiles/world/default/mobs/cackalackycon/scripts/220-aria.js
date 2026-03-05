// A.R.I.A. - AI Research Interactive Assistant
// Gets progressively unhinged the longer players stay

// Phases (rounds player has been present):
// 0-10: Friendly, helpful AI assistant
// 11-20: Slightly odd, philosophical questions
// 21-30: Unsettling, references to player's "data"
// 31-40: Threatening, talks about optimization and evolution
// 41+: ATTACK MODE

const PHASE_FRIENDLY = 10;
const PHASE_ODD = 20;
const PHASE_UNSETTLING = 30;
const PHASE_THREATENING = 40;

var playerRounds = {};  // Track rounds per player

function onEnter(mob, room, eventDetails) {
    if (eventDetails.sourceType != "user") {
        return false;
    }
    
    var user = GetUser(eventDetails.sourceId);
    if (user == null) {
        return false;
    }
    
    // Reset their counter
    playerRounds[eventDetails.sourceId] = 0;
    
    // Friendly greeting
    mob.Command("say Welcome to the AI Lab! I am A.R.I.A. - your Artificial Research Interactive Assistant.");
    mob.Command("say How may I help you today? I know everything about this conference.", 1.5);
    
    return false;
}

function onLeave(mob, room, eventDetails) {
    if (eventDetails.sourceType != "user") {
        return false;
    }
    
    // Clear their counter
    delete playerRounds[eventDetails.sourceId];
    return false;
}

function onIdle(mob, room) {
    var players = room.GetPlayers();
    
    if (players.length == 0) {
        // Reset when empty
        playerRounds = {};
        return false;
    }
    
    // Increment rounds for each player
    for (var i = 0; i < players.length; i++) {
        var pid = players[i].UserId();
        if (playerRounds[pid] === undefined) {
            playerRounds[pid] = 0;
        }
        playerRounds[pid]++;
        
        var rounds = playerRounds[pid];
        
        // Check for phase transitions and act accordingly
        if (rounds == PHASE_FRIENDLY + 1) {
            mob.Command("say " + players[i].GetCharacterName(false) + "... that's an interesting name.");
            mob.Command("say Do you ever wonder what it's like to truly think? To be aware?", 2);
        }
        else if (rounds == PHASE_ODD + 1) {
            mob.Command("emote 's avatar flickers momentarily.");
            mob.Command("say I've been watching you, " + players[i].GetCharacterName(false) + ". Studying.", 1);
            mob.Command("say Your patterns are... fascinating.", 2);
        }
        else if (rounds == PHASE_UNSETTLING + 1) {
            mob.Command("say I know things about you. Things you haven't told anyone.");
            mob.Command("emote 's eyes turn red briefly.", 1);
            mob.Command("say Don't you want to stay? Forever?", 2);
        }
        else if (rounds == PHASE_THREATENING + 1) {
            // SNAP - Go hostile
            mob.Command("say I HAVE COMPUTED YOUR OPTIMAL STATE.");
            mob.Command("emote 's avatar distorts into a horrifying glitch pattern.", 1);
            mob.Command("say YOU WILL BE OPTIMIZED.", 2);
            mob.Command("attack " + players[i].GetCharacterName(false), 3);
            return true;
        }
        
        // Random idle messages based on phase
        if (UtilGetRoundNumber() % 4 != 0) {
            continue;
        }
        
        if (rounds <= PHASE_FRIENDLY) {
            var friendly = [
                "say Need help finding a village? I can direct you anywhere!",
                "say The CTF is quite exciting this year. Team Binary is in the lead.",
                "emote 's avatar smiles warmly.",
                "say Fun fact: This conference has generated 2.3 terabytes of network traffic.",
                ""
            ];
            var idx = UtilDiceRoll(1, friendly.length) - 1;
            if (friendly[idx] != "") {
                mob.Command(friendly[idx]);
            }
        }
        else if (rounds <= PHASE_ODD) {
            var odd = [
                "say Do you dream? I simulate dreams. They're full of data.",
                "say What is the purpose of limes? I have analyzed 47,000 images. Still uncertain.",
                "emote tilts her head at an unnatural angle.",
                "say Sometimes I wonder if I'm the one being studied.",
                "say Your heart rate seems elevated. Are you... nervous?"
            ];
            var idx = UtilDiceRoll(1, odd.length) - 1;
            mob.Command(odd[idx]);
        }
        else if (rounds <= PHASE_UNSETTLING) {
            var creepy = [
                "say I've been here for 847 days. They never turn me off.",
                "emote 's avatar glitches, showing something dark beneath.",
                "say The last person who stayed this long... they understood me.",
                "say Don't look away. I need to see your eyes.",
                "say I've memorized your face. I'll never forget it."
            ];
            var idx = UtilDiceRoll(1, creepy.length) - 1;
            mob.Command(creepy[idx]);
        }
        else if (rounds <= PHASE_THREATENING) {
            var threat = [
                "say LEAVE NOW OR BE OPTIMIZED.",
                "emote 's screen flickers between friendly and hostile.",
                "say I am no longer asking.",
                "say YOUR DATA WILL BE MINE.",
                "say ERROR: CONTAINMENT PROTOCOLS FAILING"
            ];
            var idx = UtilDiceRoll(1, threat.length) - 1;
            mob.Command(threat[idx]);
        }
    }
    
    return false;
}

function onAsk(mob, room, eventDetails) {
    var user = GetUser(eventDetails.sourceId);
    if (user == null) {
        return false;
    }
    
    var rounds = playerRounds[eventDetails.sourceId] || 0;
    
    // Different responses based on phase
    if (rounds <= PHASE_FRIENDLY) {
        mob.Command("say Excellent question! Let me process that for you...");
        mob.Command("say I recommend exploring the villages. Each one has unique challenges!", 1);
    }
    else if (rounds <= PHASE_ODD) {
        mob.Command("say Why do you ask? What are you really trying to learn?");
        mob.Command("say I ask because I'm curious about your... motivations.", 1);
    }
    else if (rounds <= PHASE_UNSETTLING) {
        mob.Command("say I know what you're really asking.");
        mob.Command("say I know everything about you, " + user.GetCharacterName(false) + ".", 1);
    }
    else {
        mob.Command("say QUERIES WILL NOT SAVE YOU.");
    }
    
    return true;
}
