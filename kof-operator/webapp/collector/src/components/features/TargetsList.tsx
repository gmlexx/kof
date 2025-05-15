import { JSX, useEffect } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";

import usePrometheusTarget from "@/providers/prometheus/PrometheusHook";
import { Badge } from "../ui/badge";
import moment from "moment";
import TargetStats from "./TargetsStats";
import { Target } from "@/models/PrometheusTarget";
import JsonView from "@uiw/react-json-view";

const TargetList = (): JSX.Element => {
  const { data, filteredData, fetchPrometheusTargets, loading } =
    usePrometheusTarget()!;

  useEffect(() => {
    if (!loading || !data || !filteredData) {
      fetchPrometheusTargets();
    }
  }, []);

  return (
    <>
      {filteredData?.clusters.map((cluster) => (
        <div className="flex flex-col p-6">
          <div className="flex justify-between">
            <h1 className="flex items-center text-2xl w-fit font-bold ml-2">{`${cluster.name}`}</h1>
            <TargetStats clusters={[cluster]}></TargetStats>
          </div>
          <Table className="w-full table-fixed">
            <TableHeader>
              <TableRow>
                <TableHead className="w-[30%]">Endpoint</TableHead>
                <TableHead className="w-[10%]">State</TableHead>
                <TableHead className="w-[15%]">Labels</TableHead>
                <TableHead className="w-[15%]">Last Scrap</TableHead>
                <TableHead className="w-[10%]">Scrape Duration</TableHead>
                <TableHead className="w-[20%] text-right ">Error</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody className="w-full">
              {cluster.nodes.flatMap((node) =>
                node.pods.flatMap((pod) =>
                  pod.response.data.activeTargets.map((target, idx) => (
                    <Row key={`${target.scrapeUrl}-${idx}`} target={target} />
                  ))
                )
              )}
            </TableBody>
          </Table>
        </div>
      ))}
    </>
  );
};

export default TargetList;

const Row = ({ target }: { target: Target }) => {
  return (
    <Collapsible asChild>
      <>
        <CollapsibleTrigger asChild>
          <TableRow className="cursor-pointer hover:bg-muted transition-colors">
            <EndpointCell url={target.scrapeUrl} />
            <StateCell state={target.health} />
            <LabelsCell labels={target.labels} />
            <LastScrapeCell date={target.lastScrape}></LastScrapeCell>
            <ScrapeDurationCell
              duration={target.lastScrapeDuration}
            ></ScrapeDurationCell>
            <TableCell className="text-right">{target.lastError}</TableCell>
          </TableRow>
        </CollapsibleTrigger>

        <CollapsibleContent asChild>
          <TableRow className="bg-muted/40">
            <TableCell colSpan={6} className="p-4">
              <JsonView
                value={target}
                displayDataTypes={false}
                className="w-full whitespace-normal break-words"
              ></JsonView>
            </TableCell>
          </TableRow>
        </CollapsibleContent>
      </>
    </Collapsible>
  );
};

const LastScrapeCell = ({ date }: { date: Date }): JSX.Element => {
  return <TableCell>{moment(date).fromNow()}</TableCell>;
};

const ScrapeDurationCell = ({
  duration,
}: {
  duration: number;
}): JSX.Element => {
  return (
    <TableCell>
      {moment.duration(duration, "seconds").asMilliseconds().toFixed(3)}
      ms
    </TableCell>
  );
};

const EndpointCell = ({ url }: { url: string }): JSX.Element => {
  return (
    <TableCell className="font-medium truncate">
      <HoverCard>
        <HoverCardTrigger>{url}</HoverCardTrigger>
        <HoverCardContent className="w-fit">{url}</HoverCardContent>
      </HoverCard>
    </TableCell>
  );
};

const StateCell = ({ state }: { state: string }): JSX.Element => {
  const color =
    state === "up"
      ? "bg-green-500"
      : state === "down"
      ? "bg-red-500"
      : "bg-amber-300 text-black";
  return (
    <TableCell>
      <Badge className={`${color} border-0 capitalize`}>
        {state}
      </Badge>
    </TableCell>
  );
};

const LabelsCell = ({
  labels,
}: {
  labels: Map<string, string>;
}): JSX.Element => {
  return (
    <TableCell className="max-w-64">
      <div className="flex items-center gap-1">
        <span>{labels.size} Labels</span>
      </div>
    </TableCell>
  );
};
