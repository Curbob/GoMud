// Base16 - Registration desk NPC
// Gives free badge to new players

const BASIC_BADGE_ID = 25001;
const badgeNouns = ["badge", "register", "registration", "check in", "checkin"];

function onAsk(mob, room, eventDetails) {

    if ( (user = GetUser(eventDetails.sourceId)) == null ) {
        return false;
    }

    match = UtilFindMatchIn(eventDetails.askText, badgeNouns);
    if ( match.found ) {

        // Check if they already have any badge
        if ( user.HasItemId(25001) || user.HasItemId(25002) || user.HasItemId(25003) || user.HasItemId(25004) ) {
            mob.Command("say You've already got a badge! Head north to the con floor.");
            return true;
        }

        // Give them the badge!
        mob.Command("emote checks a list and nods.");
        mob.Command("say Found you! Here's your badge.");
        
        badge = CreateItem(BASIC_BADGE_ID);
        user.GiveItem(badge);
        
        mob.Command("say That badge is your key to everything. Guard it well.", 1);
        mob.Command("say If you want upgrades later, talk to <ansi fg=\"mobname\">Nutcrunch</ansi> at Badge Hacking.", 2);
        mob.Command("say The con floor is to the north. Have fun!", 3);
        
        return true;
    }

    return false;
}

function onGreet(mob, room, eventDetails) {

    if ( (user = GetUser(eventDetails.sourceId)) == null ) {
        return false;
    }

    // Check if they need a badge
    if ( !user.HasItemId(25001) && !user.HasItemId(25002) && !user.HasItemId(25003) && !user.HasItemId(25004) ) {
        mob.Command("say Welcome to CackalackyCon! Ask me about your <ansi fg=\"command\">badge</ansi> to get registered.");
        return true;
    }

    mob.Command("say Welcome back! The con floor is to the north.");
    return true;
}

// When player enters the room
function onEnter(mob, room, eventDetails) {

    if ( eventDetails.sourceType != "user" ) {
        return false;
    }

    if ( (user = GetUser(eventDetails.sourceId)) == null ) {
        return false;
    }

    // Only greet if they don't have a badge yet
    if ( !user.HasItemId(25001) && !user.HasItemId(25002) && !user.HasItemId(25003) && !user.HasItemId(25004) ) {
        mob.Command("wave @" + user.ShorthandId());
        mob.Command("say Hey there! Need to pick up your badge?", 1);
    }

    return false;
}

var RANDOM_IDLE = [
    "emote checks another name off the registration list.",
    "say Got your badge yet?",
    "emote hands a shiny electronic badge to a newcomer.",
    "say The badge this year is pretty special. Don't ask me how the virus thing works.",
    "emote straightens the stack of lanyards.",
    "",
    "",
];

function onIdle(mob, room) {

    if ( UtilGetRoundNumber() % 4 != 0 ) {
        return false;
    }

    randNum = UtilDiceRoll(1, RANDOM_IDLE.length) - 1;
    if ( randNum < RANDOM_IDLE.length && RANDOM_IDLE[randNum] != "" ) {
        mob.Command(RANDOM_IDLE[randNum]);
        return true;
    }

    return false;
}
