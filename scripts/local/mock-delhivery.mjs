// mock-delhivery.mjs — a tiny local stand-in for the Delhivery API, for LOCAL TESTING ONLY.
//
// Why: create-order fails closed if it cannot confirm serviceability (#23), and fulfilment
// calls Delhivery to create a shipment. When you have no Delhivery access locally, point the
// backend at this mock instead of the real API — the real backend code runs unchanged.
//
// Use:
//   node scripts/local/mock-delhivery.mjs            # listens on http://localhost:4100
//   PORT=4200 node scripts/local/mock-delhivery.mjs  # or pick another port
// Then in .env.local:
//   DELHIVERY_BASE_URL=http://localhost:4100
//
// It answers the four endpoints the backend uses:
//   GET  /c/api/pin-codes/json/        -> every pincode is serviceable (prepaid)
//   GET  /api/kinko/v1/invoice/charges/.json -> a flat shipping charge
//   GET  /api/dc/expected_tat          -> a fixed 3-day TAT
//   POST /api/cmu/create.json          -> success + a fake MOCK... waybill
// Anything else returns 404 and is logged, so you can see if the backend calls something new.

import http from "node:http";
import { URL } from "node:url";

const PORT = Number(process.env.PORT || process.argv[2] || 4100);

function send(res, status, bodyObj) {
  const body = JSON.stringify(bodyObj);
  res.writeHead(status, { "Content-Type": "application/json" });
  res.end(body);
}

// A serviceable pincode record. IsServiceable() requires: the echoed `pin` matches the
// requested pincode, `pre_paid` is "Y", and `remarks` is empty (not "Embargo").
function serviceable(pincode) {
  return {
    delivery_codes: [
      {
        postal_code: {
          pin: pincode,
          city: "Mock City",
          district: "Mock District",
          state_code: "TG",
          pre_paid: "Y",
          cash: "Y",
          pickup: "Y",
          repl: "Y",
          cod: "Y",
          is_oda: "N",
          country_code: "IN",
          remarks: "",
        },
      },
    ],
  };
}

function charges(destPin, originPin) {
  return [
    {
      actual_weight: 500,
      base_cost: 40,
      charged_weight: 500,
      cod_charges: 0,
      da_charges: 0,
      destination_pin: destPin || "",
      fuel_charges: 10,
      handling_charges: 0,
      other_charges: 0,
      origin_pin: originPin || "",
      rto_charges: 0,
      total_amount: 50,
      zone: "A",
      weight_slab: "0.5",
      status: "Success",
      charge_DL: 50,
      gross_amount: 50,
    },
  ];
}

function tat() {
  const d = new Date();
  d.setDate(d.getDate() + 3);
  return {
    msg: "success",
    data: { tat: 3, expected_delivery_date: d.toISOString().slice(0, 10) },
    success: true,
  };
}

function shipment() {
  // GetWaybill() reads packages[0].waybill; IsSuccessful() needs success && a waybill.
  const waybill = "MOCK" + Date.now().toString().slice(-10);
  return {
    success: true,
    packages: [
      {
        status: "Success",
        client: "MOCK",
        sort_code: "MCK",
        remarks: [],
        waybill,
        refnum: "MOCKREF" + waybill,
        cod_amount: "0",
        payment_mode: "Prepaid",
      },
    ],
    cash_total: 0,
    cod_total: 0,
    prepaid: 0,
    rmk: "",
  };
}

const server = http.createServer((req, res) => {
  const u = new URL(req.url, `http://localhost:${PORT}`);
  const path = u.pathname;

  if (req.method === "GET" && path === "/c/api/pin-codes/json/") {
    const pincode = (u.searchParams.get("filter_codes") || "").trim();
    console.log(`[mock-delhivery] serviceability ${pincode} -> serviceable`);
    return send(res, 200, serviceable(pincode));
  }

  if (req.method === "GET" && path === "/api/kinko/v1/invoice/charges/.json") {
    console.log(`[mock-delhivery] charges ${u.searchParams.get("d_pin")} -> Rs.50`);
    return send(res, 200, charges(u.searchParams.get("d_pin"), u.searchParams.get("o_pin")));
  }

  if (req.method === "GET" && path === "/api/dc/expected_tat") {
    console.log(`[mock-delhivery] TAT -> 3 days`);
    return send(res, 200, tat());
  }

  if (req.method === "POST" && path === "/api/cmu/create.json") {
    // Drain the body (the backend sends format=json&data=...), we don't need to parse it.
    let n = 0;
    req.on("data", (c) => (n += c.length));
    req.on("end", () => {
      const r = shipment();
      console.log(`[mock-delhivery] create shipment (${n} bytes) -> waybill ${r.packages[0].waybill}`);
      send(res, 200, r);
    });
    return;
  }

  console.log(`[mock-delhivery] UNHANDLED ${req.method} ${path} — returning 404`);
  send(res, 404, { error: "mock-delhivery: unhandled endpoint", method: req.method, path });
});

server.listen(PORT, () => {
  console.log(`[mock-delhivery] listening on http://localhost:${PORT}`);
  console.log(`[mock-delhivery] set DELHIVERY_BASE_URL=http://localhost:${PORT} in .env.local`);
});
