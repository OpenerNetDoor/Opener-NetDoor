import 'package:flutter/material.dart';

import '../models/profile_models.dart';
import '../state/app_state.dart';

class NodesScreen extends StatelessWidget {
  const NodesScreen({super.key, required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Nodes / Locations')),
      body: AnimatedBuilder(
        animation: state,
        builder: (BuildContext context, _) {
          final List<NodeLocation> nodes = state.availableNodes;
          return Column(
            children: <Widget>[
              SwitchListTile(
                title: const Text('Auto-select best node'),
                value: state.autoSelectNode,
                onChanged: (bool value) => state.setAutoSelectNode(value),
              ),
              Expanded(
                child: ListView.builder(
                  itemCount: nodes.length,
                  itemBuilder: (BuildContext context, int index) {
                    final NodeLocation node = nodes[index];
                    final bool selected = state.autoSelectNode
                        ? node.id == 'auto'
                        : state.selectedNode == node.id;
                    final bool favorite = state.favoriteNodes.contains(node.id);
                    return ListTile(
                      title: Text('${node.label} (${node.region})'),
                      subtitle: Text(node.id == 'auto' ? 'dynamic latency routing' : 'latency ${node.latencyMs} ms'),
                      leading: Icon(selected ? Icons.check_circle : Icons.radio_button_unchecked),
                      trailing: IconButton(
                        icon: Icon(favorite ? Icons.star : Icons.star_border),
                        onPressed: () => state.toggleFavoriteNode(node.id),
                      ),
                      onTap: () {
                        if (node.id == 'auto') {
                          state.setAutoSelectNode(true);
                          state.setNode('');
                          return;
                        }
                        state.setAutoSelectNode(false);
                        state.setNode(node.id);
                      },
                    );
                  },
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}
