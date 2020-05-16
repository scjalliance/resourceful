"use strict";

(function() {
	const source = new EventSource("../stream");
	const status = document.querySelector("p#status");

	// Supporting functions
	const makeRow = function(id, status, ...cells) {
		const row = document.createElement("tr");
		row.id = id;
		row.className = "status-" + status;
		for (let i = 0; i < cells.length; i++) {
			row.appendChild(cells[i]);
		}
		return row;
	}

	const updateRow = function(element, status, ...values) {
		element.className = "status-" + status;
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
	}

	const makeTimeCell = function(cn, when) {
		const t = document.createElement("time");
		t.dateTime = when;
		t.innerText = formatDuration(timeSince(when));
		const td = document.createElement("td");
		td.className = cn;
		td.appendChild(t);
		return td;
	}

	const updateCell = function(element, value) {
		if (element.children.length > 0 && element.children[0].tagName.toLowerCase() == "time") {
			const t = element.children[0];
			if (t.dateTime != value) {
				t.dateTime = value;
			}
			t.innerText = formatDuration(timeSince(t.dateTime));
		} else {
			if (element.innerText != value) {
				element.innerText = value;
			}
		}
	}

	const timeSince = function(when) {
		when = Date.parse(when);
		const now = Date.now();
		//if (now > when) {
		//	return 0;
		//}
		return now - when;
	};

	const formatDuration = function(duration) {
		const hours = Math.floor(duration / 3600000);
		duration -= hours * 3600000;
		const minutes = Math.floor(duration / 60000);
		duration -= minutes * 60000;
		const seconds = Math.floor(duration / 1000);
		return `${hours}:${('0' + minutes).slice(-2)}:${('0' + seconds).slice(-2)}`;
	}

	// Update time elements every second
	{
		let interval = 1000; // 1 second
		var updateTimes = function() {
			let times = document.querySelectorAll("time");
			for (const t of times) {
				if (t.dateTime) {
					t.innerText = formatDuration(timeSince(t.dateTime));
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

	// Lease management
	{
		const tbody = document.querySelector("section#leases table tbody");

		const hasDecayed = function(lease) {
			if (lease.decay == 0) {
				return true
			}
			// FIXME: Calculate death of leases with decay
			return false
		}

		const parseLease = function(lease) {
			return {
				"id": lease.instance.id,
				"resource": lease.resource,
				"program": lease.properties["resource.name"] || lease.properties["program.name"] || lease.resource,
				"user": lease.properties["user.account"] || lease.properties["user.id"] || lease.instance.user,
				"computer": lease.properties["host.name"] || lease.instance.host,
				"status": lease.status,
				"started": lease.properties["process.creation"] || lease.started,
				"dead": lease.status == "released" && hasDecayed(lease)
			};
		};

		const addLease = function(parent, lease) {
			let row = makeRow(
				lease.id, lease.status,
				makeCell("program", lease.program),
				makeCell("user", lease.user),
				makeCell("computer", lease.computer),
				makeCell("status", lease.status),
				makeTimeCell("time", lease.started)
			);
			row.dataset.resource = lease.resource;
			parent.appendChild(row);
		};

		const updateLease = function(element, lease) {
			updateRow(element, lease.status, lease.program, lease.user, lease.computer, lease.status, lease.started);
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

		const updateLeaseTable = function(data) {
			let existing = {};
			let found = {};

			if (data.resource) {
				existing = collectChildrenForResource(tbody, data.resource);
			}

			if (data.leases) {
				for (const rawLease of data.leases) {
					const lease = parseLease(rawLease);
					if (lease.dead) {
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

		//update(sample);

		source.addEventListener("leases", function(e) {
			const data = JSON.parse(e.data);
			console.log(data);
			//document.body.innerHTML += e.data + '<br>';
			updateLeaseTable(data);
			status.textContent = "";
		}, false);
	}
})();
