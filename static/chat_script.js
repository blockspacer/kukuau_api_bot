var storage = localStorage;

function playNotification(){
    if ($("#mute:checked").length == 0){
        var au = document.getElementById("audio-notification");
        au.play();
    }
}

var messages_updated = Math.round( Date.now() / 1000 );
var contacts_updated = Math.round( Date.now() / 1000 );


var message_for = $("#with").prop("value");

function paste_message(message){
    var text_message = "<div class='media msg'><div class='media-body'><h4 class='media-heading'>{{From}} <small class='time'>{{time}}</small></h4><div class='col-lg-11'>{{Body}}</div></div></div><hr>";
    var result = Mustache.render(text_message, message);
    $(result).insertBefore("#chat-end");
    document.getElementById( 'chat-end' ).scrollIntoView(false);
}

function update_messages(){
    data = {m_for: message_for, after:messages_updated}
    $.ajax({type:"POST",
        url:            url_prefix+"/messages",
        contentType:    'application/json',
        data:           JSON.stringify(data),
        dataType:       'json',
        success:        function(x){
            x.messages.forEach(function(message){
                paste_message(message);
            });
            messages_updated = x.next_;

        }
    });
    return true;
}


function set_contact_new_message(contact_id, count){
    var p = $("#"+contact_id),
        c_w = p.find(".new-message-counter");

    if (parseInt(c_w.attr("count")) != count){
        if (count == 0){
            c_w.text("");
        } else {
            c_w.text("("+count+")");
            playNotification();
            p.remove();
            p.insertAfter("#write-all")
        }
        c_w.attr("count", count);
    }
}

function paste_new_contact(contact){
    if (contact.NewMessagesCount != 0){
        var c_text = "<div class='contact' id='{{ID}}'><a class='a-contact' href='"+url_prefix+"?with={{ID}}'> {{Name}} <span class='small' id='s-{{ID}}' class='new-message-counter' count='{{NewMessagesCount}}'>({{NewMessagesCount}})<span></a></div>";
        var result = Mustache.render(c_text, contact);
        $(result).insertAfter("#write-all")
        playNotification();
    }
}

function update_contacts(){
    var exists = $(".contact");
    var ex_values = new Array();
    for (var k in exists){
        if (exists[k]["id"] != undefined){
            ex_values.push(exists[k]["id"]);
        }
    }
    data = {after: contacts_updated, exist:ex_values}
    $.ajax({type:"POST",
        url:            url_prefix+"/contacts",
        contentType:    'application/json',
        data:           JSON.stringify(data),
        dataType:       'json',
        success:        function(x){
            if (x.ok){
                var update = [];
                x['old'].forEach(function(c){
                    console.log("old: ",c);
                    set_contact_new_message(c.ID, c.NewMessagesCount);
                    update.push(c.ID);
                });
                x['new'].forEach(function(c){
                    console.log("new: ",c);
                    paste_new_contact(c);
                    update.push(c.ID);
                });
                $(".new-message-counter").each(function(i,el){
                    var id = el.attributes.getNamedItem("id").value.substring(2);
                    if (update.indexOf(id) == -1){
                        $(el).text("");
                        $(el).attr("count", 0);
                    }
                });
            }
        }
    });
    return true;
}

$("#chat-form").on("submit", function(e){
    e.preventDefault();
    var body = $("#chat-form-message").val(),
        from = $("#from").attr("value");
        to = $("#with").attr("value");

    console.log("body: ", body, "from: ", from, "to: ", to)
    $.ajax({
        type:           "POST",
        url:            url_prefix+"/send",
        data:           JSON.stringify({from:from, to:to, body:body}),
        dataType:       'json',
        success:        function(x){
                        console.log(x);
                        if (x.ok == true) {
                             paste_message(x.message);
                             $("#chat-form-message").val("");
                        }else{
                             window.location.href = "/chat";
                        }
        }
    });
});


function delete_chat(between){
    $.ajax({
        type:"POST",
        url:"/delete/"+between,
        dataType:"json",
        success: function(x){
            $("#removed").text(x.removed);
            $("#removed").show(500);
        }
    });
}

setInterval(function(){
    update_messages();
    update_contacts();
    return true;
}, 5000);


if($(window).width() < 600){
    var cw = $("#contacts-wrapper");
    cw.attr("closed", "true");

    cw.css(
    {'margin-left':'-200px',
    'position':'relative',
	'top':0,
	'left':0,
	'z-index':2000,
	'overflow':'visible',
	'background-color':'white'}
    )
    cw.prepend("<p class='open'><span class='glyphicon glyphicon-chevron-right'></span></p>")

    $("#contacts-wrapper p.open").css({
        'position':'absolute',
		'top':'10px',
		'left':'200px'
    })


    $("p.open").click(function(x){
        var cw = $("#contacts-wrapper"),
            closed = cw.attr("closed");
        console.log(x, cw, closed);

        if (closed == "true"){
            cw.animate({left:"200px"},400);
            cw.attr("closed", "false");
            $("#contacts-wrapper p.open span").removeClass("glyphicon-chevron-right");
            $("#contacts-wrapper p.open span").addClass("glyphicon-chevron-left");
          }else{
            cw.animate({left:'0px'},300);
            cw.attr("closed", "true");
            $("#contacts-wrapper p.open span").removeClass("glyphicon-chevron-left");
            $("#contacts-wrapper p.open span").addClass("glyphicon-chevron-right");
            }
    });

    $("a.a-contact").click(function(e){
        e.preventDefault();
        console.log(e);


        cw.animate({left:'0px'},300, function(x){
            cw.attr("closed", "true");
            $("#contacts-wrapper p.open span").removeClass("glyphicon-chevron-left");
            $("#contacts-wrapper p.open span").addClass("glyphicon-chevron-right");

            window.location.href = e.toElement.href;
        });
    })
}

var DELAY = 700, clicks = 0, timer = null;
 $("a.a-contact").on("click", function(e){
        e.preventDefault();
        clicks++;  //count clicks
        var a = this;
        if(clicks === 1) {
            timer = setTimeout(function() {
                console.log("one click");
                clicks = 0;             //after action performed, reset counter
                window.location.href = a.href;
            }, DELAY);

        } else {
            clearTimeout(timer);    //prevent single-click action
            console.log("dbl click");

            var    id = this.parentNode.attributes.getNamedItem("id").value,
                input = $("#"+id+" div.name-change"),
                name = a.text.trim();


            $(a).hide();
            input.show();
            console.log(a,input, name, id);

            clicks = 0;             //after action performed, reset counter
        }

}).on("dblclick", function(e){
        e.preventDefault();  //cancel system double-click event
});

function applyNewName(id){
            var wrapper = $('#'+id),
            a = wrapper.find('a'),
            form = wrapper.find('.name-change'),
            new_name = wrapper.find(".name-change-input").val();

        $.ajax({
            type:           "POST",
            url:            url_prefix+"/contacts_change",
            data:           JSON.stringify({'id':id, 'new_name':new_name}),
            dataType:       'json',
            success:        function(x){
                            console.log(x);
                            if (x.ok == true) {
                                a.text(new_name);
                                form.hide();
                                a.show();
                            } else {
                                console.log(x);
                            }
            }
        });

}

function notApplyNewName(id){
            var wrapper = $('#'+id),
            a = wrapper.find('a'),
            form = wrapper.find('.name-change');

            form.hide();
            a.show();
}

$(".name-change-input").keydown(function(e){
    var for_id = $(e.target).attr("name");
    if (e.keyCode == 13){
        applyNewName(for_id);
    }
    if (e.keyCode == 27){
        notApplyNewName(for_id);
    }

})

if (storage.getItem("k_audio_muted") == 'true'){
    $("#mute").prop("checked", true);
}

$("#mute").change(function() {
    if(this.checked) {
        localStorage.setItem("k_audio_muted", true);
    } else {
        localStorage.setItem("k_audio_muted", false);
    }
});

var chat_message_container = $("#chat-form-message");
chat_message_container.focus();
chat_message_container.keydown(function(e){
    if (e.ctrlKey && e.keyCode == 13) {
        $("#chat-form").submit();
    }
});

chat_message_container.focus(function(e){
        var with_id = $("#with").val();
        set_messages_read(with_id);
})


function set_messages_read(from){
    data = {from:from};
    $.ajax({type:"POST",
        url:            url_prefix+"/messages_read",
        contentType:    'application/json',
        data:           JSON.stringify(data),
        dataType:       'json',
        success:        function(x){
            if (x.ok==true){
                $("#s-"+from).text("");
            }
        }
    });
    return true;
}

var chat_end = document.getElementById( 'chat-end' );
if (chat_end != null){
    chat_end.scrollIntoView(false);
}

