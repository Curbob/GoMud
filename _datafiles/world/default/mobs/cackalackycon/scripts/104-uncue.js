// Uncue - CTF Arena
// Gives CTF fragment for participating (defeating a script kiddie)

const CTF_FRAGMENT = 5103;
const SCRIPT_KIDDIE_ID = 200;

const fragmentNouns = ["fragment", "key", "access", "server", "piece", "flag"];
const ctfNouns = ["ctf", "compete", "play", "join", "participate", "challenge"];

function onAsk(mob, room, eventDetails) {
    if ((user = GetUser(eventDetails.sourceId)) == null) {
        return false;
    }
    
    // Already has fragment?
    if (user.HasItemId(CTF_FRAGMENT)) {
        mob.Command("say You've already got your CTF fragment. Now go find the others.");
        return true;
    }
    
    // Asking about fragment
    match = UtilFindMatchIn(eventDetails.askText, fragmentNouns);
    if (match.found) {
        // Check if they've killed any script kiddies
        kills = user.GetMobKills(SCRIPT_KIDDIE_ID);
        
        if (kills > 0) {
            mob.Command("say You've taken down " + String(kills) + " script kiddies? Not bad.");
            mob.Command("say That counts as CTF participation in my book. Here.", 1);
            
            fragment = CreateItem(CTF_FRAGMENT);
            user.GiveItem(fragment);
            
            user.SendText("");
            user.SendText("<ansi fg=\"lime\">You received: CTF key fragment</ansi>");
            user.SendText("");
            
            mob.Command("say That's one of four. The old server room awaits.", 2);
            return true;
        } else {
            mob.Command("say Looking for the key fragment, huh?");
            mob.Command("say Simple rules: defend the arena. Take down at least one <ansi fg=\"mobname\">script kiddie</ansi>.", 1);
            mob.Command("say They keep spawning in here. Shouldn't be hard to find one.", 2);
            mob.Command("say Come back when you've got a kill. Then we'll talk.", 3);
            return true;
        }
    }
    
    // Asking about CTF/participating
    match2 = UtilFindMatchIn(eventDetails.askText, ctfNouns);
    if (match2.found) {
        mob.Command("say Want in on the CTF? We've got challenges running 24/7.");
        mob.Command("say But here's a side quest: clear out the <ansi fg=\"mobname\">script kiddies</ansi>.", 1);
        mob.Command("say They keep trying to brute force everything. Embarrassing.", 2);
        mob.Command("say Take one down and I'll give you something useful.", 3);
        return true;
    }
    
    return false;
}

function onGreet(mob, room, eventDetails) {
    if ((user = GetUser(eventDetails.sourceId)) == null) {
        return false;
    }
    
    if (user.HasItemId(CTF_FRAGMENT)) {
        mob.Command("say Back for more CTF? The challenges don't stop.");
        return true;
    }
    
    kills = user.GetMobKills(SCRIPT_KIDDIE_ID);
    if (kills > 0) {
        mob.Command("say Hey, you've got some script kiddie kills. Ask me about the <ansi fg=\"command\">fragment</ansi>.");
        return true;
    }
    
    return false;
}
