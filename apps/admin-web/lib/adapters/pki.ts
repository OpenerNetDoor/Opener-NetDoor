import type { NodeCertificate, PKIIssuer } from "@opener-netdoor/shared-types";

export interface IssuerVM {
  id: string;
  issuerId: string;
  source: string;
  status: string;
  activatedAt: string;
  retiredAt: string;
  createdAt: string;
}

export interface CertificateVM {
  id: string;
  serial: string;
  issuer: string;
  status: string;
  notAfter: string;
  rotatedFrom: string;
}

export function toIssuerVM(item: PKIIssuer): IssuerVM {
  return {
    id: item.id,
    issuerId: item.issuer_id,
    source: item.source,
    status: item.status,
    activatedAt: item.activated_at ?? "n/a",
    retiredAt: item.retired_at ?? "n/a",
    createdAt: item.created_at,
  };
}

export function toIssuersVM(items: PKIIssuer[]): IssuerVM[] {
  return items.map(toIssuerVM);
}

export function toCertificateVM(item: NodeCertificate): CertificateVM {
  return {
    id: item.id,
    serial: item.serial_number,
    issuer: item.issuer || item.issuer_id || item.ca_id,
    status: item.revoked_at ? "revoked" : "active",
    notAfter: item.not_after,
    rotatedFrom: item.rotate_from_cert_id ?? "n/a",
  };
}

export function toCertificatesVM(items: NodeCertificate[]): CertificateVM[] {
  return items.map(toCertificateVM);
}