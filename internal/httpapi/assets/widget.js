(function(){
  try {
    var existing = document.getElementById("mp-feedback-bubble");
    if (existing) { return; }

    var bubble = document.createElement("div");
    bubble.id = "mp-feedback-bubble";
    bubble.style.position = "fixed";
    bubble.style.right = "16px";
    bubble.style.bottom = "16px";
    bubble.style.width = "56px";
    bubble.style.height = "56px";
    bubble.style.borderRadius = "28px";
    bubble.style.boxShadow = "0 4px 16px rgba(0,0,0,0.2)";
    bubble.style.background = "#0d6efd";
    bubble.style.cursor = "pointer";
    bubble.style.display = "flex";
    bubble.style.alignItems = "center";
    bubble.style.justifyContent = "center";
    bubble.style.zIndex = "2147483647";
    bubble.style.color = "#fff";
    bubble.style.fontSize = "28px";
    bubble.style.userSelect = "none";
    bubble.setAttribute("aria-label","Send feedback");
    bubble.innerText = "💬";
    document.body.appendChild(bubble);

    var panel = document.createElement("div");
    panel.id = "mp-feedback-panel";
    panel.style.position = "fixed";
    panel.style.right = "16px";
    panel.style.bottom = "80px";
    panel.style.width = "320px";
    panel.style.maxWidth = "92vw";
    panel.style.background = "#ffffff";
    panel.style.border = "1px solid rgba(0,0,0,0.1)";
    panel.style.boxShadow = "0 8px 24px rgba(0,0,0,0.2)";
    panel.style.borderRadius = "12px";
    panel.style.padding = "12px";
    panel.style.fontFamily = "system-ui, -apple-system, Segoe UI, Roboto, Ubuntu, Cantarell, Noto Sans, sans-serif";
    panel.style.display = "none";
    panel.style.zIndex = "2147483647";

    var headline = document.createElement("div");
    headline.style.fontWeight = "600";
    headline.style.marginBottom = "8px";
    headline.innerText = "Send feedback";
    panel.appendChild(headline);

    var contact = document.createElement("input");
    contact.type = "text";
    contact.placeholder = "Email or phone";
    contact.autocomplete = "email";
    contact.style.width = "100%";
    contact.style.margin = "6px 0";
    contact.style.padding = "10px";
    contact.style.border = "1px solid #ced4da";
    contact.style.borderRadius = "8px";
    panel.appendChild(contact);

    var message = document.createElement("textarea");
    message.placeholder = "Your message";
    message.rows = 4;
    message.style.width = "100%";
    message.style.margin = "6px 0 8px";
    message.style.padding = "10px";
    message.style.border = "1px solid #ced4da";
    message.style.borderRadius = "8px";
    panel.appendChild(message);

    var send = document.createElement("button");
    send.type = "button";
    send.innerText = "Send";
    send.style.width = "100%";
    send.style.padding = "10px 12px";
    send.style.border = "0";
    send.style.borderRadius = "8px";
    send.style.background = "#0d6efd";
    send.style.color = "#fff";
    send.style.fontWeight = "600";
    send.style.cursor = "pointer";
    panel.appendChild(send);

    var status = document.createElement("div");
    status.style.marginTop = "6px";
    status.style.fontSize = "12px";
    status.style.minHeight = "16px";
    panel.appendChild(status);

    document.body.appendChild(panel);

    bubble.addEventListener("click", function(){
      panel.style.display = (panel.style.display === "none" ? "block" : "none");
    });

    function show(msg, ok){
      status.innerText = msg;
      status.style.color = ok ? "#157347" : "#dc3545";
    }

    function validate(){
      var contactValue = (contact.value || "").trim();
      var messageValue = (message.value || "").trim();
      if (contactValue.length < 3) { show("Please enter a valid email or phone.", false); return null; }
      if (messageValue.length === 0) { show("Please write a message.", false); return null; }
      return {contact: contactValue, message: messageValue};
    }

    send.addEventListener("click", function(){
      var valid = validate();
      if (!valid) { return; }
      send.disabled = true;
      show("Sending...", true);

      var payload = JSON.stringify({
        site_id: "{{ .SiteID }}",
        contact: valid.contact,
        message: valid.message
      });

      var endpoint = (location.protocol + "//" + location.host + "/api/feedback");
      try {
        var scriptTag = document.currentScript || (function(){
          var candidates = document.querySelectorAll('script[src*="widget.js"]');
          return candidates[candidates.length - 1];
        })();
        if (scriptTag && scriptTag.src) {
          var link = document.createElement("a");
          link.href = scriptTag.src;
          endpoint = link.protocol + "//" + link.host + "/api/feedback";
        }
      } catch(e){}

      fetch(endpoint, {
        method: "POST",
        headers: {"Content-Type": "application/json"},
        body: payload,
        keepalive: true
      }).then(function(resp){
        if (!resp.ok) { throw new Error("HTTP " + resp.status); }
        return resp.json();
      }).then(function(){
        show("Thanks! Sent.", true);
        contact.value = "";
        message.value = "";
        send.disabled = false;
      }).catch(function(err){
        show("Failed to send. Please try again.", false);
        send.disabled = false;
        console.error(err);
      });
    });
  } catch(e) {
    console.error(e);
  }
})();
