// Fish heal - tiny (3 HP)
function onTrigger(actor, triggersLeft) {
    var healAmt = actor.AddHealth(3);
    if (healAmt > 0) {
        SendUserMessage(actor.UserId(), 'The fish restores <ansi fg="healing">'+String(healAmt)+' health</ansi>.');
    }
}
