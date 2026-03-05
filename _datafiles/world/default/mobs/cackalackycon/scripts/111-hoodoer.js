// Hoodoer - Fancy Hacker Swan Host
// Multicolored tux, top hat, says Cheerio constantly

function onIdle(mob, room) {
    if (UtilGetRoundNumber() % 4 != 0) {
        return false;
    }
    
    var actions = [
        "say Cheerio!",
        "say Cheerio, good hackers!",
        "say I say, cheerio!",
        "emote adjusts their magnificent top hat.",
        "emote twirls their cane elegantly.",
        "say Absolutely smashing party, what what!",
        "emote 's tuxedo shifts from pink to electric blue.",
        "say Cheerio! Lovely to see you!",
        "emote raises a champagne glass. Cheerio!",
        "say Pip pip! Cheerio!",
        "emote bows with theatrical flourish.",
        "say One does so enjoy a proper Hacker Swan, cheerio!",
        "",
        ""
    ];
    
    var idx = UtilDiceRoll(1, actions.length) - 1;
    if (actions[idx] != "") {
        mob.Command(actions[idx]);
        return true;
    }
    
    return false;
}

function onGreet(mob, room, eventDetails) {
    var user = GetUser(eventDetails.sourceId);
    if (user == null) return false;
    
    mob.Command("say Cheerio, " + user.GetCharacterName(false) + "! Welcome to the Hacker Swan!");
    mob.Command("emote tips their magnificent top hat.", 1);
    mob.Command("say I do hope you're having a simply SMASHING time, what what!", 2);
    
    return true;
}

function onAsk(mob, room, eventDetails) {
    var user = GetUser(eventDetails.sourceId);
    if (user == null) return false;
    
    var ask = eventDetails.askText.toLowerCase();
    
    if (ask.includes("cheerio")) {
        mob.Command("say Cheerio! Cheerio indeed! It's the only proper greeting!");
        mob.Command("say Cheerio cheerio cheerio!", 1);
        return true;
    }
    
    if (ask.includes("tux") || ask.includes("suit") || ask.includes("hat")) {
        mob.Command("emote 's tuxedo ripples through several colors proudly.");
        mob.Command("say Ah yes, the attire! Custom made by a very confused tailor!", 1);
        mob.Command("say Cheerio!", 2);
        return true;
    }
    
    if (ask.includes("swan") || ask.includes("party") || ask.includes("ball")) {
        mob.Command("say The Hacker Swan is a TRADITION! Elegance meets excellence!");
        mob.Command("say W00zles and I host it every year. Cheerio!", 1.5);
        return true;
    }
    
    mob.Command("say Splendid question! The answer is, of course... Cheerio!");
    return true;
}
