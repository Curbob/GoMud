
// Invoked once every round if mob is idle
function onIdle(mob, room) {

    var roundNow = UtilGetRoundNumber();

    var charmedUserId = mob.GetCharmedUserId();

    if ( charmedUserId == 0 ) {
        return false;
    }

    var charmer = GetUser(charmedUserId);

    // If charmer isn't in world, skip
    if ( charmer == null ) {
        return false;
    }

    // If they aren't in the same room for some reason, skip
    if ( charmer.GetRoomId() != mob.GetRoomId() ) {

        var lostRoundCt = mob.GetTempData("roundsLost");
        if (lostRoundCt == null ) lostRoundCt = 0;

        lostRoundCt++;

        if ( lostRoundCt >= 3 ) {
            
            mob.SetTempData('roundsLost', 0);

            var targetRoom = GetRoom(charmer.GetRoomId() );
            if ( targetRoom != null ) {
                mob.MoveRoom( charmer.GetRoomId() );
                room.SendText("A large " + UtilApplyColorPattern('swirling portal', 'green') + " appears, and " + mob.GetCharacterName(true) + " steps into it, right before it disappears.");
                targetRoom.SendText("A large " + UtilApplyColorPattern('swirling portal', 'green') + " appears, and " + mob.GetCharacterName(true) + " steps out of it, right before it disappears.");
                mob.Command("saytoonly @" + charmer.UserId() + " I almost lost you " + charmer.GetCharacterName(true) + "!");
            }
            
            return true;
        }
        
        mob.SetTempData('roundsLost', lostRoundCt);

        return false;

    }
    
    var lastTipRound = mob.GetTempData("lastTipRound");
    if ( lastTipRound == null ) {
        lastTipRound = 0;
    }

    if ( lastTipRound == -1 ) {
        return true;
    }

    lastUserInput = charmer.GetLastInputRound();
    var roundsSinceInput = roundNow - lastUserInput;

    // Only give a tip if the user has been inactive for 5 rounds
    if ( roundsSinceInput < 3 ) {
        return true;
    }

    roundsPassed = roundNow - lastTipRound;

    // give at least 5 rounds between tips, even if the user remains inactive.
    if ( roundsPassed < 5 ) {
        return false;
    }

    // Check if we're in CackalackyCon zone (rooms 2000-2999)
    var roomId = room.RoomId();
    var inConZone = (roomId >= 2000 && roomId < 3000);

    if (inConZone) {
        // CackalackyCon-specific tips
        switch( UtilDiceRoll(1, 8) ) {
            case 1:
                mob.Command("saytoonly @" + charmer.UserId() + " Welcome to <ansi fg=\"green-bold\">CackalackyCon</ansi>! Explore the villages and meet some fellow hackers.");
                break;
            case 2:
                mob.Command("saytoonly @" + charmer.UserId() + " Check out the <ansi fg=\"room-title\">Lockpick Village</ansi> if you want to practice your picking skills!");
                break;
            case 3:
                mob.Command("saytoonly @" + charmer.UserId() + " The <ansi fg=\"room-title\">CTF Arena</ansi> has challenges if you're feeling competitive.");
                break;
            case 4:
                mob.Command("saytoonly @" + charmer.UserId() + " Talk to the NPCs around here - they might have quests or useful info.");
                break;
            case 5:
                mob.Command("saytoonly @" + charmer.UserId() + " You can find help on many subjects by typing <ansi fg=\"command\">help</ansi>.");
                break;
            case 6:
                mob.Command("saytoonly @" + charmer.UserId() + " Rumor has it there's a hidden server room somewhere in this hotel...");
                break;
            case 7:
                mob.Command("saytoonly @" + charmer.UserId() + " Head to the <ansi fg=\"room-title\">Bar</ansi> to grab a drink and hear some gossip.");
                break;
            case 8:
                mob.Command("saytoonly @" + charmer.UserId() + " Something strange is going on at this con. Keep your eyes open.");
                break;
        }
    } else {
        // Fantasy world tips (original)
        switch( UtilDiceRoll(1, 10) ) {
            case 1:
                if ( charmer.GetStatPoints() > 0 ) {
                    mob.Command("saytoonly @" + charmer.UserId() + " It looks like you've got some stat points to spend. Type <ansi fg=\"command\">status train</ansi> to upgrade your stats!");
                }
                break;
            case 2:
                if ( !charmer.HasQuest("4-start") ) {
                    mob.Command("saytoonly @" + charmer.UserId() + " There's a guard in the barracks that constantly complains about being hungry. You should <ansi fg=\"command\">ask</ansi> him about it.");
                }
                break;
            case 3:
                if ( !charmer.HasQuest("2-start") ) {
                    mob.Command("saytoonly @" + charmer.UserId() + " I have heard the king worries. If we can find an audience with him we can try to <ansi fg=\"command\">ask</ansi> him about a quest. He is north of town square.");
                }
                break;
            case 4:
                mob.Command("saytoonly @" + charmer.UserId() + " You can find help on many subjects by typing <ansi fg=\"command\">help</ansi>.");
                break;
            case 5:
                mob.Command("saytoonly @" + charmer.UserId() + " I can create a portal to take us back to <ansi fg=\"room-title\">Town Square</ansi> any time. Just <ansi fg=\"command\">ask</ansi> me about it.");
                break;
            case 6:
                mob.Command("saytoonly @" + charmer.UserId() + " If you have friends to play with, you can party up! <ansi fg=\"command\">help party</ansi> to learn more.");
                break;
            case 7:
                mob.Command("saytoonly @" + charmer.UserId() + " You can send a message to everyone using the <ansi fg=\"command\">broadcast</ansi> command.");
                break;
            case 8:
                if ( charmer.GetLevel() < 2 ) {
                    mob.Command("saytoonly @" + charmer.UserId() + " There are some <ansi fg=\"mobname\">rats</ansi> to bash around the <ansi fg=\"room-title\">The Sanctuary of the Benevolent Heart</ansi> south of <ansi fg=\"room-title\">Town Square</ansi>. Just don't go TOO far south, it gets dangerous!");
                }
                break;
            case 9:
                mob.Command("saytoonly @" + charmer.UserId() + " Type <ansi fg=\"command\">status</ansi> to learn about yourself!");
                break;
            case 10:
                mob.Command("saytoonly @" + charmer.UserId() + " Killing stuff is a great way to get stronger, but don't pick a fight with the locals!");
                break;
        }
    }

    // Prevent from triggering too often
    mob.SetTempData("lastTipRound", roundNow);

    return true;
}


// Things to ask to get a portal created
var homeNouns = ["home", "portal", "return", "townsquare", "town square", "entrance", "registration"];

// Things to ask to shut up the guide
var silenceNouns = ["silence", "quiet", "shut up", "shh"];

var leaveNouns = ["leave", "leave me alone", "die", "quit", "go away", "unfollow", "get lost"];

function onAsk(mob, room, eventDetails) {

    var charmedUserId = mob.GetCharmedUserId();

    if ( eventDetails.sourceId != charmedUserId ) {
        return false;
    }

    if ( (user = GetUser(eventDetails.sourceId)) == null ) {
        return false;
    }

    match = UtilFindMatchIn(eventDetails.askText, homeNouns);
    if ( match.found ) {

        // Check if in con zone
        var roomId = room.RoomId();
        var inConZone = (roomId >= 2000 && roomId < 3000);
        var homeRoom = inConZone ? 2010 : 0; // 2010 is CackalackyCon Bar, 0 is Town Square
        var homeName = inConZone ? "the Bar" : "Town Square";

        if ( user.GetRoomId() == homeRoom || (homeRoom == 0 && user.GetRoomId() == 1) ) {
            mob.Command("saytoonly @"+String(eventDetails.sourceId)+" we're already at <ansi fg=\"room-title\">" + homeName + "</ansi>. <ansi fg=\"command\">Look</ansi> around!");
            return true;
        }

        mob.Command("saytoonly @"+String(eventDetails.sourceId)+" back to <ansi fg=\"room-title\">" + homeName + "</ansi>? Sure thing, lets go!");
        mob.Command("emote whispers a soft incantation and summons a " + UtilApplyColorPattern("glowing portal", "green") + ".");

        room.AddTemporaryExit("glowing portal", ":green", homeRoom, 3);

        return true;
    }

    match = UtilFindMatchIn(eventDetails.askText, silenceNouns);
    if ( match.found ) {

        mob.Command("saytoonly @"+String(eventDetails.sourceId)+" I'll try and be quieter.");
        
        mob.SetTempData("lastTipRound", -1);

        return true;
    }


    match = UtilFindMatchIn(eventDetails.askText, leaveNouns);
    if ( match.found ) {

        mob.Command("saytoonly @"+String(eventDetails.sourceId)+" I'll be on my way then.");
        mob.Command("emote bows and bids you farewell, disappearing into the scenery");
        mob.Command("despawn charmed mob expired");

        return true;
    }

    return false;
}
