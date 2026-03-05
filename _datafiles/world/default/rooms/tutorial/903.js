
const allowed_commands = ["help", "broadcast", "look", "status", "inventory", "experience", "conditions", "equip"];
const teach_commands = ["get cap", "equip cap", "portal"];
const teacherMobId = 57;
const teacherName = "Lime of Learning";
const capItemId = 20043;
const newbieKitItemId = 150; // hacker kit for CackalackyCon

var commandNow = 0; // Which command they are on



// Generic Command Handler
function onCommand(cmd, rest, user, room) {
    
    ignoreCommand = false;

    teacherMob = getTeacher(room);

    fullCommand = ExpandCommand(cmd);
    if ( rest.length > 0 ) {
        fullCommand = cmd + ' ' + rest;
    }

    if ( commandNow >= 2 ) {
        return false;
    }
    
    if ( teach_commands[commandNow] == fullCommand ) {
        
        if ( fullCommand == "equip cap" ) {
            teacherMob.Command("say Good job!", 1.0);
        } else {
            teacherMob.Command("say Good job! You earned it!", 1.0);
        }

        commandNow++;

    } else {

        if ( allowed_commands.includes(cmd) || teach_commands.slice(0, commandNow).includes(cmd) ) {
            return false;
        }
        
        ignoreCommand = true;
    }

    switch (commandNow) {
        case 0:
            teacherMob.Command('emote gestures to the <ansi fg="item">graduation cap</ansi> on the ground.', 1.0);
            teacherMob.Command('say type <ansi fg="command">get cap</ansi> to pick up the <ansi fg="item">graduation cap</ansi>.', 1.0);
            break;
        case 1:
            teacherMob.Command('say Go ahead and wear the <ansi fg="item">graduation cap</ansi> by typing <ansi fg="command">equip cap</ansi>.', 1.0);
            break;
        case 2:

            teacherMob.Command('say It\'s time to say goodbye!', 1.0);
            teacherMob.Command('say I\'ll summon a portal to send you to the <ansi fg="zone">CackalackyCon Main Entrance</ansi>, where your adventure begins.', 1.0);

            exits = room.GetExits();
            if ( !exits.portal ) {
                teacherMob.Command('emote glows intensely, and a ' + UtilApplyColorPattern('lime-green portal', 'glowing') + ' appears!', 1.0);
                room.AddTemporaryExit('lime-green portal', ':green', 2000, "1 real day"); // RoomId 2000 is CackalackyCon Main Entrance
            }

            teacherMob.Command('say Enter the portal by typing <ansi fg="command">lime-green portal</ansi> (or just <ansi fg="command">portal</ansi>) when you are ready.', 1.0);
            teacherMob.Command('say See you at the con! Remember: when in doubt, look for limes.', 1.0);
            
            break;
        default:
            break;
    }
    
    return ignoreCommand;
}




// If there is no book here, add the book item
function onEnter(user, room) {
    
    teacherMob = getTeacher(room);
    clearGroundItems(room);
    
    sendWorkingCommands(user);

    itm = CreateItem(capItemId);
    teacherMob.GiveItem(itm);

    itm2 = CreateItem(newbieKitItemId);
    user.GiveItem(itm2);

    teacherMob.Command('emote appears in a ' + UtilApplyColorPattern("flash of light!", "glowing"));

    teacherMob.Command('say Congratulations on completing the CackalackyCon orientation!', 1.0);
    teacherMob.Command('drop cap', 1.0);
    teacherMob.Command('emote gestures to the <ansi fg="item">graduation cap</ansi> on the ground.', 3.0);
    teacherMob.Command('say type <ansi fg="command">get cap</ansi> to pick up the <ansi fg="item">graduation cap</ansi>.', 1.0);

    return true;
}

function onExit(user , room) {
    // Destroy the guide (cleanup)
    destroyTeacher(room);
    
    canGoSouth = false;
    commandNow = 0;
}

function onLoad(room) {
    canGoSouth = false;
    commandNow = 0;
}

function getTeacher(room) {
    var mobActor = room.GetMob(teacherMobId, true);
    mobActor.SetCharacterName(teacherName);
    return mobActor;
}

function destroyTeacher(room) {
    var mobActor = room.GetMob(teacherMobId);
    if ( mobActor != null ) {
        mobActor.Command(`suicide vanish`);
    } 
}

function sendWorkingCommands(user) {

    ac = [];
    unlockedCommands = teach_commands.slice(0, commandNow);

    for (var i in allowed_commands ) {
        ac.push(allowed_commands[i]);
    }
    
    for ( i in unlockedCommands ) {
        ac.push(unlockedCommands[i]);
    }
    
    user.SendText("");
    user.SendText("");
    user.SendText('    <ansi fg="red">NOTE:</ansi> Most commands have been <ansi fg="203">DISABLED</ansi> and <ansi fg="203">WILL NOT WORK</ansi> until you <ansi fg="51">COMPLETE THIS TUTORIAL</ansi>!');
    //user.SendText('          The commands currently available are: <ansi fg="command">'+ac.join('</ansi>, <ansi fg="command">')+'</ansi>');
    user.SendText("");
    user.SendText("");

}

function clearGroundItems(room) {

    allGroundItems = room.GetItems();
    for ( var i in allGroundItems ) {
        room.DestroyItem(allGroundItems[i]);
    }

}