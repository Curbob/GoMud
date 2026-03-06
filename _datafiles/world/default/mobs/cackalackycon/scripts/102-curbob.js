// Curbob - The 20-year con veteran
// Always ready with a story from cons past

const STORIES = [
    "Let me tell you about the great badge hack of 2012. Someone figured out the badges were transmitting unencrypted. Within an hour, badges were rickrolling each other across the hotel.",
    "CarolinaCon 2008. The hotel's network went down, so we built our own. Thirty hackers, a pile of routers, and a lot of caffeine. IT guy found us at 3am and just... joined in.",
    "There was this one year — 2015 I think — where someone brought an actual payphone. Fully functional. Nobody knows how they got it there.",
    "The CTF in '11 was a bloodbath. Final challenge came down to two teams. Both solved it at the exact same second. We had to check server logs. Drama for MONTHS.",
    "First time they served limes at the bar was 2018. Started as a joke. Someone said 'this con needs more vitamin C.' Now it's tradition.",
    "I remember when you could fit everyone in the con in a single hotel room. Those 3am discussions were something else.",
    "2009. Tesla coil incident. We don't talk about the Tesla coil incident. ...okay fine, it was AWESOME until the fire alarm.",
    "The lockpick village used to just be one guy named Dave with a toolbox. Now look at it. Dave would be proud. RIP Dave.",
    "Back when it was still CarolinaCon, we had a speaker no-show. So some random attendee gave a talk on lock impressioning. Best talk of the con.",
    "2016 badge had a hidden game. Took the community three months to find all the easter eggs. The designer watched us struggle and said NOTHING.",
    "There was a year where someone set up a rogue cell tower in the parking lot. Just to see what would connect. A lot. A lot connected.",
    "I've seen three marriage proposals at this con. Two of them went well.",
    "The hotel bar knows us by now. They stock extra limes starting in January.",
    "Someone once asked me what my favorite con year was. I've been thinking about that question for six years. I still don't have an answer.",
    "Twenty years. I've watched kids become professionals, professionals become legends, and legends become... well, some of them are still around. Older. Wiser. Still can't fix printers."
];

var lastStoryIndex = -1;

function onAsk(mob, room, eventDetails) {
    var user = GetUser(eventDetails.sourceId);
    if (user == null) {
        return false;
    }
    
    var question = eventDetails.askText.toLowerCase();
    
    // Story triggers
    if (question.includes("story") || question.includes("stories") || 
        question.includes("old days") || question.includes("remember") ||
        question.includes("history") || question.includes("past") ||
        question.includes("years") || question.includes("before")) {
        
        tellStory(mob, user);
        return true;
    }
    
    // Schedule/talk info
    if (question.includes("schedule") || question.includes("talk") || 
        question.includes("next") || question.includes("speaker")) {
        mob.Command("say The schedule? I know it by heart. But honestly, the best stuff happens in the hallways.");
        mob.Command("say Head to LobbyCon if you want the REAL talks.", 2);
        return true;
    }
    
    // About himself
    if (question.includes("you") || question.includes("curbob") || 
        question.includes("yourself") || question.includes("who")) {
        mob.Command("say Me? I'm nobody special. Just a guy who's been coming here for twenty years.");
        mob.Command("say Started back when it was still CarolinaCon. Different name, same spirit.", 2);
        mob.Command("say I've seen a lot. Done a lot. Mostly just watched talks and drank bad coffee.", 3);
        return true;
    }
    
    // Limes
    if (question.includes("lime") || question.includes("limes")) {
        mob.Command("say Ah, the limes. Everyone asks about the limes.");
        mob.Command("say Started in 2018. Just a joke at first. Someone put limes on everything.", 2);
        mob.Command("say Now it's tradition. Don't question tradition.", 3);
        return true;
    }
    
    // Default - tell a story anyway
    tellStory(mob, user);
    return true;
}

function tellStory(mob, user) {
    // Pick a random story, but not the same as last time
    var idx;
    do {
        idx = UtilDiceRoll(1, STORIES.length) - 1;
    } while (idx == lastStoryIndex && STORIES.length > 1);
    
    lastStoryIndex = idx;
    
    mob.Command("emote leans back with a nostalgic look.");
    mob.Command("say " + STORIES[idx], 1);
}

function onIdle(mob, room) {
    // Very occasionally prompt for stories
    if (UtilGetRoundNumber() % 50 == 0) {
        mob.Command("say Anyone want to hear about the old days? Just ask.");
    }
    return false;
}
