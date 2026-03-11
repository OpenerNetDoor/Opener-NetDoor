import { redirect } from "next/navigation";

export default function LegacyPKIIssuersRoute() {
  redirect("/pki/issuers");
}
