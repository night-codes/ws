(function (factory) {
	if (typeof define === 'function' && define.amd) {
		define(["exports"], factory);
	} else if (typeof exports === 'object') {
		factory(exports);
	} else {
		factory('undefined' !== typeof global ? global : window);
	}
}(function (exports) {
	"use strict";
	if (typeof exports === 'undefined') {
		exports = {};
	}


	var channels = {};

	function createWebSocket(path) {
		return new WebSocket((location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + path);
	}


	function Channel(url) {
		var global = 'undefined' !== typeof global ? global : window;
		var document = global.document;
		var sock = null;
		var prevID = 0;
		var requestTimeout = 30;
		var self = this;		
		var waitOk = {};

		var cid = "" + (Math.random().toFixed(16).substring(2) + new Date().valueOf()) + url;

		function on(type, fn) {
			document.addEventListener(cid + type, function (e) {
				if (e && e.result) {
					fn(e.result);
				} else {
					fn();
				}
			});
		}
		function one(type, fn) {
			var cb = function (e) {
				e.target.removeEventListener(cid + type, cb);
				if (e && e.result) {
					fn(e.result);
				} else {
					fn();
				}
			};
			document.addEventListener(cid + type, cb);
		}

		function trigger(type, data) {
			var t = cid + type;
			var event;

			if (document.dispatchEvent) {
				if (typeof CustomEvent === "function") {
					event = new CustomEvent(t, { "result": data });
				} else if (typeof Event === "function") {
					event = new Event(t, { "result": data });
				} else if (document.createEvent) {
					event = document.createEvent('HTMLEvents');
					event.initEvent(t, true, true);
				}
				if (event) {
					event.result = data;
					document.dispatchEvent(event);
					return;
				}
			}

			event = document.createEventObject();
			event.result = data;
			document.fireEvent('on' + t, event);
			return;
		}

		(function connect() {
			function done(result) {
				try {
					result = JSON.parse(result);
				} catch (err) {
					return;
				}

				if (result && result.command) {
					if (result.requestID > 0) {
						trigger("request:" + result.command + ":" + result.requestID, result.data);
					} else {
						trigger("read:" + result.command, result);
					}
					trigger("came", result.command);
					waitOk[result.command] = true;
				}
			}

			sock = createWebSocket(url);
			sock.onopen = function () {
				trigger('wsConnect');
			};
			sock.onclose = function (e) {
				setTimeout(connect, 300);
			};
			sock.onmessage = function (e) {
				if (e && typeof e.data === 'string' || e.data instanceof Blob) {
					if (e.data instanceof Blob) { // извлекаем бинарные данные
						var reader = new FileReader();
						reader.onload = function () {
							done(reader.result);
						};
						reader.readAsText(e.data);
					} else {
						done(e.data);
					}
				}
			};
		}());

		// send message to socket and get answer to callback(data, err)
		self.send = function (command, msg, requestID) {
			command = command.replace(/\:/g, '_');
			requestID = requestID || 0;
			msg = JSON.stringify(msg);
			if (global.dev) {
				msg = [requestID, command, msg].join(":");
			} else {
				try {
					msg = new Blob([requestID, ":", command, ":", msg]);
				} catch (e) {
					if (e.name == "InvalidStateError") {
						msg = [requestID, command, msg].join(":");
						var bb = new MSBlobBuilder();
						bb.append(msg);
						msg = bb.getBlob('text/csv;charset=utf-8');
					}
				}
			}
			if (sock && sock.readyState === WebSocket.OPEN) {
				sock.send(msg);
			} else {
				one('wsConnect', function () {
					sock.send(msg);
				});
			}
		};


		// server messages handler
		self.read = function (command, callback) {
			on("read:" + command, function (result) {
				callback(result.data, function (msg) {
					if (result.srvRequestID) {
						self.send(command, msg, result.srvRequestID);
					}
				});
			});
		};

		// send request to server and wait answer to handler
		self.request = function (command, msg, callback, timeout) {
			var requestID = 0;
			command = command.replace(/\:/g, '_');

			if (typeof msg === "function") {
				timeout = callback;
				callback = msg;
				msg = "";
			}

			if (callback) {
				requestID = ++prevID;
				timeout = timeout || requestTimeout;
				one("request:" + command + ":" + requestID, function (data) {
					if (data instanceof Error) {
						callback(undefined, data);
						return;
					}
					callback(data, undefined);
				});
				if (timeout > 0) {
					setTimeout(function () {
						trigger("request:" + command + ":" + requestID, new Error("\"" + command + "\" request timeout"));
					}, timeout);
				}
			}
			self.send(command, msg, requestID);
		};

		// set request timeout
		self.setRequestTimeout = function (timeout) {
			requestTimeout = timeout;
		};

		// повесить обработчик на сообщения, санкционированные сервером (без запроса)
		self.subscribe = function (command) {
			if (sock && sock.readyState === WebSocket.OPEN) {
				self.send("subscribe", command);
			}
			on('wsConnect', function () {
				self.send("subscribe", command);
			});
		};

		self.wait = function (commands, callback) {
			var cmds = {};
			commands.forEach(function (command) {
				if (!waitOk[command]) {
					cmds[command] = true;
				}
			});

			function wt(command) {
				if (command && cmds[command]) {
					delete cmds[command];
				}
				if (Object.keys(cmds).length < 1) {
					setTimeout(callback, 1);
					return;
				}
				one('came', wt);
			}
			wt();
		};
	}


	exports.getChannel = function (url) {
		if (typeof url !== 'string') {
			return null;
		}

		if (!channels[url]) {
			channels[url] = new Channel(url);
		}

		return channels[url];
	};

	return exports;

}));
