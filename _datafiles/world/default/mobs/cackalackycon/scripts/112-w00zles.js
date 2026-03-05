// W00zles - Fancy Hacker Swan Host
// Rainbow sequined tux, top hat, says Cheerio constantly

function onIdle(mob, room) {
    if (UtilGetRoundNumber() % 4 != 0) {
        return false;
    }
    
    var actions = [
        "say Cheerio!",
        "say Cheerio, darlings!",
        "say Cheerio, one and all!",
        "emote 's sequined top hat sparkles magnificently.",
        "emote glides across the floor with impossible grace.",
        "say Rather splendid gathering, cheerio!",
        "emote 's rainbow tuxedo catches the light dramatically.",
        "say Cheerio! Delightful to make your acquaintance!",
        "emote produces a tiny cucumber sandwich from nowhere.",
        "say Tally ho and cheerio!",
        "emote curtsies with elaborate formality.",
        "say The Hacker Swan is the pinnacle of sophistication, cheerio!",
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
    
    mob.Command("say Cheerio, dear " + user.GetCharacterName(false) + "!");
    mob.Command("emote sweeps off their sequined top hat in greeting.", 1);
    mob.Command("say Hoodoer and I are DELIGHTED you could join us! Cheerio!", 2);
    
    return true;
}

function onAsk(mob, room, eventDetails) {
    var user = GetUser(eventDetails.sourceId);
    if (user == null) return false;
    
    var ask = eventDetails.askText.toLowerCase();
    
    if (ask.includes("cheerio")) {
        mob.Command("say Cheerio! 'Tis the queen's greeting! Or king's! Or sovereign's!");
        mob.Command("say One simply MUST say cheerio! Cheerio!", 1);
        return true;
    }
    
    if (ask.includes("tux") || ask.includes("suit") || ask.includes("hat") || ask.includes("sequin")) {
        mob.Command("emote does a small spin, sequins exploding with color.");
        mob.Command("say Every sequin was placed by hand! By MY hands! Cheerio!", 1);
        return true;
    }
    
    if (ask.includes("hoodoer")) {
        mob.Command("say Hoodoer? My co-host! My partner in elegance! My cheerio companion!");
        mob.Command("say We've been hosting the Swan together for YEARS. Cheerio!", 1.5);
        return true;
    }
    
    if (ask.includes("food") || ask.includes("drink") || ask.includes("sandwich")) {
        mob.Command("emote produces another tiny cucumber sandwich.");
        mob.Command("say Cucumber sandwiches! The only proper refreshment! Cheerio!", 1);
        return true;
    }
    
    mob.Command("say Mmm yes, quite! The answer is undoubtedly... Cheerio!");
    return true;
}
