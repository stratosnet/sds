var isLoginPage = false
var API = {

    token: "",
    init: function() {
        this.token = localStorage.getItem("token")
        if (this.token == null && !isLoginPage) {
            this._redirect("/login.html")
        }
    },

    login: function() {
        _this = this
        data = {}
        data.username = $('input[name="username"]').val()
        data.password = $('input[name="password"]').val()
        this.call("POST", "/login", data, function(response) {
            if (response.errcode == 200 && response.data.token != "") {
                localStorage.setItem("token", response.data.token)
                _this._redirect("/")
            }
        })
    },

    Logout: function() {
        this.token = null
        localStorage.removeItem("token")
        this.init()
    },

    call: function(method, url, data, callback) {
        switch (method) {
            case "POST":
                this._post(url, data, callback)
                break;
            case "DELETE":
                this._delete(url, data, callback)
                break;
            default:
                this._get(url, data, callback)
                break;

        }
    },

    isLogined: function() {
        this.token = localStorage.getItem("token")
        if (this.token != null && isLoginPage) {
            this._redirect("/")
            return true
        }
        return false
    },

    _redirect: function(url) {
        window.location.href = url
    },

    _post: function(url, data, callback) {
        $.ajax({ type: "post", url: url, async: false, // 使用同步方式
            data: JSON.stringify(data),
            contentType: "application/json",
            dataType: "json",
            success: callback,
        });
    },

    _get: function(url, data, callback) {
        $.ajax({ type: "get", url: url, async: false, // 使用同步方式
            data: data,
            dataType: "json",
            success: callback,
        });
    },

    _delete: function(url, data, callback) {
        $.ajax({ type: "delete", url: url, async: false, // 使用同步方式
            data: JSON.stringify(data),
            contentType: "application/json",
            dataType: "json",
            success: callback,
        });
    },

    param: function(name) {
        var results = new RegExp('[\?&]' + name + '=([^]*)').exec(window.location.href);
        if (results==null){
            return null;
        }
        else{
            return results[1] || 0;
        }
    }
}

function tips(dom, type, msg)
{
    alert = '<div class="alert alert-%type% alert-dismissible fade show" role="alert">%data%<button type="button" class="close" data-dismiss="alert" aria-label="Close"><span aria-hidden="true">&times;</span></button></div>'
    alert = alert.replace("%type%", type)
    alert = alert.replace("%data%", msg)
    $(dom).html(alert)
    $(".alert").alert()
    setInterval(function() {
        $(".alert").alert("close")
    }, 1000)
}

$(function() {
    API.init()

    $("#logout").click(function() {
        API.Logout()
    })
})