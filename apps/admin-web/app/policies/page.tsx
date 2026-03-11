import { redirect } from "next/navigation";

export default function PoliciesIndexPage() {
  redirect("/policies/tenants");
}
