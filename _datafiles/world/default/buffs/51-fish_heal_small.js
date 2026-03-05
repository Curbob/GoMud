// Fish heal - small (5 HP)
function onTrigger(actor, triggersLeft) {
    var healAmt = actor.AddHealth(5);
    if (healAmt > 0) {
        SendUserMessage(actor.UserId(), 'The fish restores <ansi fg="healing">'+String(healAmt)+' health</ansi>.');
    }
}
