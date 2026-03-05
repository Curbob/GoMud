// s0lray - Hardware Hacking Village
// Gives hardware fragment when player asks or completes task

const HARDWARE_FRAGMENT = 5101;
const IOT_BOARD = 5007;  // Dropped by rogue IoT devices

const fragmentNouns = ["fragment", "key", "access", "server", "old machine", "piece"];
const helpNouns = ["help", "task", "quest", "do", "earn"];

function onAsk(mob, room, eventDetails) {
    if ((user = GetUser(eventDetails.sourceId)) == null) {
        return false;
    }
    
    // Already has fragment?
    if (user.HasItemId(HARDWARE_FRAGMENT)) {
        mob.Command("say You've already got my fragment. Good luck with the others.");
        return true;
    }
    
    // Asking about the fragment/key
    match = UtilFindMatchIn(eventDetails.askText, fragmentNouns);
    if (match.found) {
        // Check if they have an IoT board to trade
        if (user.HasItemId(IOT_BOARD)) {
            mob.Command("say Ooh, is that an IoT board? Those are hard to get.");
            mob.Command("say Tell you what - <ansi fg=\"command\">give board to s0lray</ansi> and I'll give you something special.", 1);
        } else {
            mob.Command("say Looking for access to the old server room, huh?");
            mob.Command("say I've got a piece of the key, but I need something in return.", 1);
            mob.Command("say Bring me a <ansi fg=\"itemname\">circuit board</ansi> from one of those rogue IoT devices.", 2);
            mob.Command("say They drop them when defeated. Come back when you've got one.", 3);
        }
        return true;
    }
    
    // Asking for help/task
    match2 = UtilFindMatchIn(eventDetails.askText, helpNouns);
    if (match2.found) {
        mob.Command("say Want to help? Hunt down some <ansi fg=\"mobname\">rogue IoT devices</ansi>.");
        mob.Command("say They're in the IoT Playground. Bring me a <ansi fg=\"itemname\">circuit board</ansi>.", 1);
        return true;
    }
    
    return false;
}

function onGive(mob, room, eventDetails) {
    if (eventDetails.sourceType == "mob") {
        return false;
    }
    
    if ((user = GetUser(eventDetails.sourceId)) == null) {
        return false;
    }
    
    // Already has fragment?
    if (user.HasItemId(HARDWARE_FRAGMENT)) {
        mob.Command("say You've already got my fragment!");
        mob.Command("drop !" + String(eventDetails.item.ItemId), 1);
        return true;
    }
    
    // Check if it's the IoT board
    if (eventDetails.item && eventDetails.item.ItemId == IOT_BOARD) {
        mob.Command("say Perfect! This is exactly what I needed.");
        mob.Command("emote examines the circuit board with obvious delight.", 1);
        mob.Command("say Here's your end of the deal.", 2);
        
        // Give them the fragment
        fragment = CreateItem(HARDWARE_FRAGMENT);
        user.GiveItem(fragment);
        
        user.SendText("");
        user.SendText("<ansi fg=\"lime\">You received: hardware key fragment</ansi>");
        user.SendText("");
        
        mob.Command("say That fragment is one of four. Good luck finding the others.", 3);
        return true;
    }
    
    // Wrong item
    mob.Command("say Interesting, but not what I need.");
    mob.Command("say Bring me a <ansi fg=\"itemname\">circuit board</ansi> from a rogue IoT device.", 1);
    mob.Command("drop !" + String(eventDetails.item.ItemId), 2);
    return true;
}
