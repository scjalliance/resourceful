"use strict";

(function() {
	const source = new EventSource("../stream");
	const status = document.querySelector("p#status");

	// Supporting functions
	const makeRow = function(id, ...cells) {
		const row = document.createElement("tr");
		row.id = id;
		for (let i = 0; i < cells.length; i++) {
			row.appendChild(cells[i]);
		}
		return row;
	};

	const updateRow = function(element, ...values) {
		for (let i = 0; i < values.length; i++) {
			const child = element.children[i];
			if (!child) {
				continue;
			}
			updateCell(child, values[i]);
		}
	};

	const makeCell = function(cn, text) {
		const td = document.createElement("td");
		td.className = cn;
		td.innerText = text;
		return td;
	};

	const makeTimeSinceCell = function(cn, when) {
		const t = document.createElement("time");
		t.dateTime = when;
		t.classList.add("since");
		t.innerText = formatTimeSince(when);
		const td = document.createElement("td");
		td.classList.add(cn);
		td.appendChild(t);
		return td;
	};

	const makeTimeUntilCell = function(cn, when) {
		const t = document.createElement("time");
		t.dateTime = when;
		t.classList.add("until");
		t.innerText = formatTimeUntil(when);
		const td = document.createElement("td");
		td.classList.add(cn);
		td.appendChild(t);
		return td;
	};

	const updateCell = function(element, value) {
		if (element.children.length > 0 && element.children[0].tagName.toLowerCase() == "time") {
			const t = element.children[0];
			if (t.dateTime != value) {
				t.dateTime = value;
			}
			if (value) {
				if (t.classList.contains("since")) {
					t.innerText = formatTimeSince(value);
				} else if (t.classList.contains("until")) {
					t.innerText = formatTimeUntil(value);
				}
			} else {
				t.innerText = "";
			}
		} else {
			if (element.innerText != value) {
				element.innerText = value;
			}
		}
	};

	const timeSince = function(when) {
		when = Date.parse(when);
		const now = Date.now();
		if (now < when) {
			return 0;
		}
		return now - when;
	};
	
	const timeUntil = function(when) {
		when = Date.parse(when);
		const now = Date.now();
		if (when < now) {
			return 0;
		}
		return when - now;
	};

	const formatTimeSince = function(when) {
		let duration = timeSince(when);
		if (duration == 0) {
			return "";
		}
		return formatDuration(duration);
	};

	const formatTimeUntil = function(when) {
		let duration = timeUntil(when);
		if (duration == 0) {
			return "";
		}
		return formatDuration(duration);
	};

	const nanoToMilli = function(duration) {
		return duration / 1000000;
	};

	const calculateConsumed = function(strategy, stats) {
		if (!stats) {
			return 0;
		}
		switch (strategy) {
			case "consumer":
				if (!stats.consumer) {
					return 0;
				}
				return stats.consumer.consumed;
			case "instance":
				if (!stats.instance) {
					return 0;
				}
				return stats.instance.consumed;
			default:
				return 0;
		}
	}

	const formatDuration = function(duration) {
		const hours = Math.floor(duration / 3600000);
		duration -= hours * 3600000;
		const minutes = Math.floor(duration / 60000);
		duration -= minutes * 60000;
		const seconds = Math.floor(duration / 1000);
		return `${hours}:${('0' + minutes).slice(-2)}:${('0' + seconds).slice(-2)}`;
	};

	// Update time elements every second
	{
		let interval = 1000; // 1 second
		var updateTimes = function() {
			let times = document.querySelectorAll("time");
			for (const t of times) {
				if (t.dateTime) {
					if (t.classList.contains("since")) {
						t.innerText = formatTimeSince(t.dateTime);
					} else if (t.classList.contains("until")) {
						t.innerText = formatTimeUntil(t.dateTime);
					}
				}
			}
		}
		setInterval(updateTimes, interval);
	}

	// Connection status handling
	{
		source.addEventListener("error", function(e) {
			status.textContent = "Connection Failure";
		}, false);
	}


	// Policy management
	{
		const tbody = document.querySelector("section#policies table tbody");

		const parsePolicy = function(policy) {
			let program = "";
			if (policy.properties) {
				program = policy.properties["resource.name"] || policy.properties["program.name"] || policy.resource;
			} else {
				program = policy.resource;
			}
			return {
				"id": "policy-" + policy.resource,
				"resource": policy.resource,
				"program": program,
				"strategy": policy.strategy || "instance",
				"limit": policy.limit
			};
		};

		const addPolicy = function(parent, policy) {
			let row = makeRow(
				policy.id,
				makeCell("program", policy.program),
				makeCell("consumed", ""),
				makeCell("available", ""),
				makeCell("total", String(policy.limit))
			);
			row.dataset.resource = policy.resource;
			row.dataset.program = policy.program;
			row.dataset.strategy = policy.strategy;
			row.dataset.limit = policy.limit;
			parent.appendChild(row);
		};

		const updatePolicy = function(row, policy) {
			row.dataset.program = policy.program;
			row.dataset.strategy = policy.strategy;
			row.dataset.limit = policy.limit;

			if (row.dataset.stats) {
				let consumed = calculateConsumed(policy.strategy, row.dataset.stats);
				updateRow(row, policy.program, String(consumed), String(policy.limit - consumed), String(policy.limit));
			} else {
				updateRow(row, policy.program, "", "", String(policy.limit));
			}
		};

		const collectChildren = function(parent) {
			let m = {};
			if (parent.children) {
				for (const child of parent.children) {
					if (child.id) {
						m[child.id] = true;
					}
				}
			}
			return m;
		};

		const updatePolicyTable = function(data) {
			let existing = collectChildren(tbody);
			let found = {};

			if (data.policies) {
				for (const rawPolicy of data.policies) {
					const policy = parsePolicy(rawPolicy);
					found[policy.id] = true;
					let row = document.getElementById(policy.id);
					if (!row) {
						addPolicy(tbody, policy);
					} else {
						updatePolicy(row, policy);
					}
				}
			}

			for (const id in existing) {
				if (!found[id]) {
					document.getElementById(id).remove();
				}
			}

			status.textContent = "";
		}

		// Listen for policy updates
		{
			source.addEventListener("policies", function(e) {
				const data = JSON.parse(e.data);
				console.log(data);
				status.textContent = "";
				updatePolicyTable(data);
			}, false);
		}
	}

	// Lease management
	{
		const tbody = document.querySelector("section#leases table tbody");

		const timeOfDeath = function(status, duration, decay, renewed, released) {
			duration = nanoToMilli(duration);
			decay = nanoToMilli(decay);
			renewed = Date.parse(renewed);
			released = Date.parse(released);
			switch (status) {
				case "active":
					return (new Date(renewed + duration + decay)).toISOString();
				case "released":
					return (new Date(released + decay)).toISOString();
				default:
					return "";
			}
		}

		const parseLease = function(lease) {
			return {
				"id": lease.instance.id,
				"pid": lease.properties["process.id"],
				"resource": lease.resource,
				"program": lease.properties["resource.name"] || lease.properties["program.name"] || lease.resource,
				"user": lease.properties["user.account"] || lease.properties["user.id"] || lease.instance.user,
				"computer": lease.properties["host.name"] || lease.instance.host,
				"status": lease.status,
				"started": lease.properties["process.creation"] || lease.started,
				"released": lease.released,
				"death": timeOfDeath(lease.status, lease.duration, lease.decay, lease.renewed, lease.released),
			};
		};

		const addLease = function(parent, lease) {
			let row = makeRow(
				lease.id,
				makeCell("program", lease.program),
				makeCell("user", lease.user),
				makeCell("computer", lease.computer),
				makeCell("pid", lease.pid),
				makeCell("status", lease.status),
				makeTimeSinceCell("time", lease.started),
				makeTimeUntilCell("remaining", lease.death)
			);
			row.className = "status-" + status;
			row.dataset.resource = lease.resource;
			row.dataset.death = lease.death;
			parent.appendChild(row);
		};

		const updateLease = function(row, lease) {
			row.className = "status-" + lease.status;
			row.dataset.death = lease.death;
			updateRow(row, lease.program, lease.user, lease.computer, lease.pid, lease.status, lease.started, lease.death);
		};

		const collectChildrenForResource = function(parent, resource) {
			let m = {};
			for (const child of parent.children) {
				if (child.id && child.dataset.resource == resource) {
					m[child.id] = true;
				}
			}
			return m;
		};

		const collectChildrenForDeath = function(parent, now) {
			let pending = [];
			for (const child of parent.children) {
				if (child.dataset.death && Date.parse(child.dataset.death) <= now) {
					pending.push(child);
				}
			}
			return pending;
		};

		const updateLeaseTable = function(data) {
			let existing = {};
			let found = {};

			if (data.resource) {
				existing = collectChildrenForResource(tbody, data.resource);
			}

			if (data.leases) {
				for (const rawLease of data.leases) {
					const lease = parseLease(rawLease);
					if (lease.death && lease.death < Date.now()) {
						console.log("dead lease detected: " + lease.id);
						continue;
					}
					found[lease.id] = true;
					let row = document.getElementById(lease.id);
					if (!row) {
						addLease(tbody, lease);
					} else {
						updateLease(row, lease);
					}
				}
			}

			for (const id in existing) {
				if (!found[id]) {
					document.getElementById(id).remove();
				}
			}

			status.textContent = "";
		}

		const updatePolicy = function(row, stats) {
			row.dataset.stats = stats;
			if (row.dataset.strategy) {
				let consumed = calculateConsumed(row.dataset.strategy, stats);
				updateRow(row, row.dataset.program, String(consumed), String(row.dataset.limit - consumed), String(row.dataset.limit));
			}
		};

		const updatePolicyTable = function(data) {
			if (data.resource && data.stats) {
				let row = document.getElementById("policy-" + data.resource);
				if (row) {
					updatePolicy(row, data.stats);
				}
			}
		}

		// Listen for lease updates
		{
			let seen = {};
			source.addEventListener("leases", function(e) {
				const data = JSON.parse(e.data);
				console.log(data);
				status.textContent = "";
				if (data.resource && data.revision) {
					let last = seen[data.resource]
					if (last && last > data.revision) {
						return
					}
					seen[data.resource] = data.revision
				}
				updatePolicyTable(data);
				updateLeaseTable(data);
			}, false);
		}

		// Scan for dead rows every second
		{
			let interval = 1000; // 1 second
			var cullDead = function() {
				let pending = collectChildrenForDeath(tbody, Date.now());
				for (const row of pending) {
					row.remove();
				}
			}
			setInterval(cullDead, interval);
		}
	}
})();
